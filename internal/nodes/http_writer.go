package nodes

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs"
	"github.com/TeaOSLab/EdgeNode/internal/caches"
	"github.com/TeaOSLab/EdgeNode/internal/remotelogs"
	"github.com/TeaOSLab/EdgeNode/internal/utils"
	"github.com/iwind/TeaGo/lists"
	"net"
	"net/http"
	"strings"
)

// HTTPWriter 响应Writer
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

	cacheWriter  caches.Writer // 缓存写入
	cacheStorage caches.StorageInterface

	isOk bool // 是否完全成功
}

// NewHTTPWriter 包装对象
func NewHTTPWriter(req *HTTPRequest, httpResponseWriter http.ResponseWriter) *HTTPWriter {
	return &HTTPWriter{
		req:    req,
		writer: httpResponseWriter,
	}
}

// Reset 重置
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

// Gzip 设置Gzip
func (this *HTTPWriter) Gzip(config *serverconfigs.HTTPGzipConfig) {
	this.gzipConfig = config
}

// Prepare 准备输出
func (this *HTTPWriter) Prepare(size int64, status int) {
	this.statusCode = status

	this.prepareGzip(size)
	this.prepareCache(size)
}

// Raw 包装前的原始的Writer
func (this *HTTPWriter) Raw() http.ResponseWriter {
	return this.writer
}

// Header 获取Header
func (this *HTTPWriter) Header() http.Header {
	if this.writer == nil {
		return http.Header{}
	}
	return this.writer.Header()
}

// AddHeaders 添加一组Header
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

// Write 写入数据
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

		// 写入缓存
		if this.cacheWriter != nil {
			_, err = this.cacheWriter.Write(data)
			if err != nil {
				_ = this.cacheWriter.Discard()
				this.cacheWriter = nil
				remotelogs.Error("HTTP_WRITER", "write cache failed: "+err.Error())
			}
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
				remotelogs.Error("HTTP_WRITER", err.Error())
			}
		} else {
			this.body = append(this.body, data...)
		}
	}
	return
}

// WriteString 写入字符串
func (this *HTTPWriter) WriteString(s string) (n int, err error) {
	return this.Write([]byte(s))
}

// SentBodyBytes 读取发送的字节数
func (this *HTTPWriter) SentBodyBytes() int64 {
	return this.sentBodyBytes
}

// WriteHeader 写入状态码
func (this *HTTPWriter) WriteHeader(statusCode int) {
	if this.writer != nil {
		this.writer.WriteHeader(statusCode)
	}
	this.statusCode = statusCode
}

// StatusCode 读取状态码
func (this *HTTPWriter) StatusCode() int {
	if this.statusCode == 0 {
		return http.StatusOK
	}
	return this.statusCode
}

// SetBodyCopying 设置拷贝Body数据
func (this *HTTPWriter) SetBodyCopying(b bool) {
	this.bodyCopying = b
}

// BodyIsCopying 判断是否在拷贝Body数据
func (this *HTTPWriter) BodyIsCopying() bool {
	return this.bodyCopying
}

// Body 读取拷贝的Body数据
func (this *HTTPWriter) Body() []byte {
	return this.body
}

// HeaderData 读取Header二进制数据
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

// SetOk 设置成功
func (this *HTTPWriter) SetOk() {
	this.isOk = true
}

// Close 关闭
func (this *HTTPWriter) Close() {
	// gzip writer
	if this.gzipWriter != nil {
		if this.bodyCopying && this.gzipBodyWriter != nil {
			_ = this.gzipBodyWriter.Close()
			this.body = this.gzipBodyBuffer.Bytes()
		}
		_ = this.gzipWriter.Close()
		this.gzipWriter = nil
	}

	// cache writer
	if this.cacheWriter != nil {
		if this.isOk {
			err := this.cacheWriter.Close()
			if err == nil {
				this.cacheStorage.AddToList(&caches.Item{
					Type:       this.cacheWriter.ItemType(),
					Key:        this.cacheWriter.Key(),
					ExpiredAt:  this.cacheWriter.ExpiredAt(),
					HeaderSize: this.cacheWriter.HeaderSize(),
					BodySize:   this.cacheWriter.BodySize(),
					Host:       this.req.Host,
					ServerId:   this.req.Server.Id,
				})
			}
		} else {
			_ = this.cacheWriter.Discard()
		}
	}
}

// Hijack Hijack
func (this *HTTPWriter) Hijack() (conn net.Conn, buf *bufio.ReadWriter, err error) {
	hijack, ok := this.writer.(http.Hijacker)
	if ok {
		return hijack.Hijack()
	}
	return
}

// Flush Flush
func (this *HTTPWriter) Flush() {
	flusher, ok := this.writer.(http.Flusher)
	if ok {
		flusher.Flush()
	}
}

// 准备Gzip
func (this *HTTPWriter) prepareGzip(size int64) {
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
		remotelogs.Error("HTTP_WRITER", err.Error())
		return
	}

	// body copy
	if this.bodyCopying {
		this.gzipBodyBuffer = bytes.NewBuffer([]byte{})
		this.gzipBodyWriter, err = gzip.NewWriterLevel(this.gzipBodyBuffer, int(this.gzipConfig.Level))
		if err != nil {
			remotelogs.Error("HTTP_WRITER", err.Error())
		}
	}

	header := this.writer.Header()
	header.Set("Content-Encoding", "gzip")
	header.Set("Transfer-Encoding", "chunked")
	header.Set("Vary", "Accept-Encoding")
	header.Del("Content-Length")
}

// 准备缓存
func (this *HTTPWriter) prepareCache(size int64) {
	if this.writer == nil {
		return
	}

	// 不支持Range
	if len(this.Header().Get("Content-Range")) > 0 {
		return
	}

	cachePolicy := sharedNodeConfig.HTTPCachePolicy
	if cachePolicy == nil || !cachePolicy.IsOn {
		return
	}

	cacheRef := this.req.cacheRef
	if cacheRef == nil || !cacheRef.IsOn {
		return
	}

	// 如果允许 ChunkedEncoding，就无需尺寸的判断，因为此时的 size 为 -1
	if !cacheRef.AllowChunkedEncoding && size < 0 {
		return
	}
	if size >= 0 && ((cacheRef.MaxSizeBytes() > 0 && size > cacheRef.MaxSizeBytes()) ||
		(cachePolicy.MaxSizeBytes() > 0 && size > cachePolicy.MaxSizeBytes())) {
		return
	}

	// 检查状态
	if len(cacheRef.Status) > 0 && !lists.ContainsInt(cacheRef.Status, this.StatusCode()) {
		return
	}

	// Cache-Control
	if len(cacheRef.SkipResponseCacheControlValues) > 0 {
		cacheControl := this.writer.Header().Get("Cache-Control")
		if len(cacheControl) > 0 {
			values := strings.Split(cacheControl, ",")
			for _, value := range values {
				if cacheRef.ContainsCacheControl(strings.TrimSpace(value)) {
					return
				}
			}
		}
	}

	// Set-Cookie
	if cacheRef.SkipResponseSetCookie && len(this.writer.Header().Get("Set-Cookie")) > 0 {
		return
	}

	// 校验其他条件
	if cacheRef.Conds != nil && cacheRef.Conds.HasResponseConds() && !cacheRef.Conds.MatchResponse(this.req.Format) {
		return
	}

	// 打开缓存写入
	storage := caches.SharedManager.FindStorageWithPolicy(cachePolicy.Id)
	if storage == nil {
		return
	}
	this.cacheStorage = storage
	life := cacheRef.LifeSeconds()
	if life <= 60 { // 最小不能少于1分钟
		life = 60
	}
	expiredAt := utils.UnixTime() + life
	cacheWriter, err := storage.OpenWriter(this.req.cacheKey, expiredAt, this.StatusCode())
	if err != nil {
		if !caches.CanIgnoreErr(err) {
			remotelogs.Error("HTTP_WRITER", "write cache failed: "+err.Error())
		}
		return
	}
	this.cacheWriter = cacheWriter
	if this.gzipWriter != nil {
		this.cacheWriter = caches.NewGzipWriter(this.cacheWriter, this.req.cacheKey, expiredAt)
	}

	// 写入Header
	for k, v := range this.Header() {
		for _, v1 := range v {
			_, err = cacheWriter.WriteHeader([]byte(k + ":" + v1 + "\n"))
			if err != nil {
				remotelogs.Error("HTTP_WRITER", "write cache failed: "+err.Error())
				_ = this.cacheWriter.Discard()
				this.cacheWriter = nil
				return
			}
		}
	}
}
