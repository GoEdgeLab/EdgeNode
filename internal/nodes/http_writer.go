package nodes

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs"
	"github.com/iwind/TeaGo/logs"
	"net"
	"net/http"
	"strings"
)

// 响应Writer
type HTTPWriter struct {
	req    *HTTPRequest
	writer http.ResponseWriter

	gzipConfig *serverconfigs.HTTPGzipConfig
	gzipWriter *gzip.Writer

	statusCode    int
	sentBodyBytes int64

	bodyCopying    bool
	body           []byte
	gzipBodyBuffer *bytes.Buffer // 当使用gzip压缩时使用
	gzipBodyWriter *gzip.Writer  // 当使用gzip压缩时使用
}

// 包装对象
func NewHTTPWriter(req *HTTPRequest, httpResponseWriter http.ResponseWriter) *HTTPWriter {
	return &HTTPWriter{
		req:    req,
		writer: httpResponseWriter,
	}
}

// 重置
func (this *HTTPWriter) Reset(httpResponseWriter http.ResponseWriter) {
	this.writer = httpResponseWriter

	this.gzipConfig = nil
	this.gzipWriter = nil

	this.statusCode = 0
	this.sentBodyBytes = 0

	this.bodyCopying = false
	this.body = nil
	this.gzipBodyBuffer = nil
	this.gzipBodyWriter = nil
}

// 设置Gzip
func (this *HTTPWriter) Gzip(config *serverconfigs.HTTPGzipConfig) {
	this.gzipConfig = config
}

// 准备输出
func (this *HTTPWriter) Prepare(size int64) {
	if this.gzipConfig == nil || this.gzipConfig.Level <= 0 {
		return
	}

	// 判断Accept是否支持gzip
	if !strings.Contains(this.req.requestHeader("Accept-Encoding"), "gzip") {
		return
	}

	// 尺寸和类型
	if size < this.gzipConfig.MinBytes() || (this.gzipConfig.MaxBytes() > 0 && size > this.gzipConfig.MaxBytes()) {
		return
	}

	// 校验其他条件
	if this.gzipConfig.Conds != nil {
		if len(this.gzipConfig.Conds.Groups) > 0 {
			if !this.gzipConfig.Conds.MatchRequest(this.req.Format) || !this.gzipConfig.Conds.MatchResponse(this.req.Format) {
				return
			}
		} else {
			// 默认校验文档类型
			contentType := this.writer.Header().Get("Content-Type")
			if len(contentType) > 0 && (!strings.HasPrefix(contentType, "text/") && !strings.HasPrefix(contentType, "application/")) {
				return
			}
		}
	}

	// 如果已经有编码则不处理
	if len(this.writer.Header().Get("Content-Encoding")) > 0 {
		return
	}

	// gzip writer
	var err error = nil
	this.gzipWriter, err = gzip.NewWriterLevel(this.writer, int(this.gzipConfig.Level))
	if err != nil {
		logs.Error(err)
		return
	}

	// body copy
	if this.bodyCopying {
		this.gzipBodyBuffer = bytes.NewBuffer([]byte{})
		this.gzipBodyWriter, err = gzip.NewWriterLevel(this.gzipBodyBuffer, int(this.gzipConfig.Level))
		if err != nil {
			logs.Error(err)
		}
	}

	header := this.writer.Header()
	header.Set("Content-Encoding", "gzip")
	header.Set("Transfer-Encoding", "chunked")
	header.Set("Vary", "Accept-Encoding")
	header.Del("Content-Length")
}

// 包装前的原始的Writer
func (this *HTTPWriter) Raw() http.ResponseWriter {
	return this.writer
}

// 获取Header
func (this *HTTPWriter) Header() http.Header {
	if this.writer == nil {
		return http.Header{}
	}
	return this.writer.Header()
}

// 添加一组Header
func (this *HTTPWriter) AddHeaders(header http.Header) {
	if this.writer == nil {
		return
	}
	for key, value := range header {
		if key == "Connection" {
			continue
		}
		for _, v := range value {
			this.writer.Header().Add(key, v)
		}
	}
}

// 写入数据
func (this *HTTPWriter) Write(data []byte) (n int, err error) {
	if this.writer != nil {
		if this.gzipWriter != nil {
			n, err = this.gzipWriter.Write(data)
		} else {
			n, err = this.writer.Write(data)
		}
		if n > 0 {
			this.sentBodyBytes += int64(n)
		}
	} else {
		if n == 0 {
			n = len(data) // 防止出现short write错误
		}
	}
	if this.bodyCopying {
		if this.gzipBodyWriter != nil {
			_, err := this.gzipBodyWriter.Write(data)
			if err != nil {
				logs.Error(err)
			}
		} else {
			this.body = append(this.body, data...)
		}
	}
	return
}

// 写入字符串
func (this *HTTPWriter) WriteString(s string) (n int, err error) {
	return this.Write([]byte(s))
}

// 读取发送的字节数
func (this *HTTPWriter) SentBodyBytes() int64 {
	return this.sentBodyBytes
}

// 写入状态码
func (this *HTTPWriter) WriteHeader(statusCode int) {
	if this.writer != nil {
		this.writer.WriteHeader(statusCode)
	}
	this.statusCode = statusCode
}

// 读取状态码
func (this *HTTPWriter) StatusCode() int {
	if this.statusCode == 0 {
		return http.StatusOK
	}
	return this.statusCode
}

// 设置拷贝Body数据
func (this *HTTPWriter) SetBodyCopying(b bool) {
	this.bodyCopying = b
}

// 判断是否在拷贝Body数据
func (this *HTTPWriter) BodyIsCopying() bool {
	return this.bodyCopying
}

// 读取拷贝的Body数据
func (this *HTTPWriter) Body() []byte {
	return this.body
}

// 读取Header二进制数据
func (this *HTTPWriter) HeaderData() []byte {
	if this.writer == nil {
		return nil
	}

	resp := &http.Response{}
	resp.Header = this.Header()
	if this.statusCode == 0 {
		this.statusCode = http.StatusOK
	}
	resp.StatusCode = this.statusCode
	resp.ProtoMajor = 1
	resp.ProtoMinor = 1

	resp.ContentLength = 1 // Trick：这样可以屏蔽Content-Length

	writer := bytes.NewBuffer([]byte{})
	_ = resp.Write(writer)
	return writer.Bytes()
}

// 关闭
func (this *HTTPWriter) Close() {
	if this.gzipWriter != nil {
		if this.bodyCopying && this.gzipBodyWriter != nil {
			_ = this.gzipBodyWriter.Close()
			this.body = this.gzipBodyBuffer.Bytes()
		}
		_ = this.gzipWriter.Close()
		this.gzipWriter = nil
	}
}

// Hijack
func (this *HTTPWriter) Hijack() (conn net.Conn, buf *bufio.ReadWriter, err error) {
	hijack, ok := this.writer.(http.Hijacker)
	if ok {
		return hijack.Hijack()
	}
	return
}

// Flush
func (this *HTTPWriter) Flush() {
	flusher, ok := this.writer.(http.Flusher)
	if ok {
		flusher.Flush()
	}
}
