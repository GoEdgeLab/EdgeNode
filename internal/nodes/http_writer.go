// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package nodes

import (
	"bufio"
	"bytes"
	"compress/flate"
	"compress/gzip"
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs"
	"github.com/TeaOSLab/EdgeNode/internal/caches"
	"github.com/TeaOSLab/EdgeNode/internal/compressions"
	"github.com/TeaOSLab/EdgeNode/internal/remotelogs"
	"github.com/TeaOSLab/EdgeNode/internal/utils"
	"github.com/andybalholm/brotli"
	_ "github.com/biessek/golang-ico"
	"github.com/iwind/TeaGo/lists"
	"github.com/iwind/TeaGo/types"
	"github.com/iwind/gowebp"
	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/webp"
	"image"
	"image/gif"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net"
	"net/http"
	"path/filepath"
	"strings"
	"sync/atomic"
)

// 限制WebP能够同时使用的Buffer内存使用量
const webpMaxBufferSize int64 = 1_000_000_000
const webpSuffix = "@GOEDGE_WEBP"

var webpTotalBufferSize int64 = 0
var webpBufferPool = utils.NewBufferPool(1024)

// HTTPWriter 响应Writer
type HTTPWriter struct {
	req    *HTTPRequest
	writer http.ResponseWriter

	size int64

	webpIsEncoding        bool
	webpBuffer            *bytes.Buffer
	webpIsWriting         bool
	webpOriginContentType string
	webpOriginEncoding    string // gzip

	compressionConfig *serverconfigs.HTTPCompressionConfig
	compressionWriter compressions.Writer
	compressionType   serverconfigs.HTTPCompressionType

	statusCode    int
	sentBodyBytes int64

	bodyCopying           bool
	body                  []byte
	compressionBodyBuffer *bytes.Buffer       // 当使用压缩时使用
	compressionBodyWriter compressions.Writer // 当使用压缩时使用

	cacheWriter  caches.Writer // 缓存写入
	cacheStorage caches.StorageInterface

	isOk       bool // 是否完全成功
	isFinished bool // 是否已完成
}

// NewHTTPWriter 包装对象
func NewHTTPWriter(req *HTTPRequest, httpResponseWriter http.ResponseWriter) *HTTPWriter {
	return &HTTPWriter{
		req:    req,
		writer: httpResponseWriter,
	}
}

// SetCompression 设置内容压缩配置
func (this *HTTPWriter) SetCompression(config *serverconfigs.HTTPCompressionConfig) {
	this.compressionConfig = config
}

// Prepare 准备输出
// 缓存不调用此函数
func (this *HTTPWriter) Prepare(size int64, status int) (delayHeaders bool) {
	this.size = size
	this.statusCode = status

	if status == http.StatusOK {
		this.prepareWebP(size)

		if this.webpIsEncoding {
			delayHeaders = true
		}
	}

	this.prepareCache(size)

	// 在WebP模式下，压缩暂不可用
	if !this.webpIsEncoding {
		this.PrepareCompression(size)
	}

	// 是否限速写入
	if this.req.web != nil &&
		this.req.web.RequestLimit != nil &&
		this.req.web.RequestLimit.IsOn &&
		this.req.web.RequestLimit.OutBandwidthPerConnBytes() > 0 {
		this.writer = NewHTTPRateWriter(this.writer, this.req.web.RequestLimit.OutBandwidthPerConnBytes())
	}

	return
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

// DeleteHeader 删除Header
func (this *HTTPWriter) DeleteHeader(name string) {
	this.writer.Header().Del(name)
}

// SetHeader 设置Header
func (this *HTTPWriter) SetHeader(name string, values []string) {
	this.writer.Header()[name] = values
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
	n = len(data)

	if this.writer != nil {
		if this.webpIsEncoding && !this.webpIsWriting {
			this.webpBuffer.Write(data)
		} else {
			// 写入压缩
			var n1 int
			if this.compressionWriter != nil {
				n1, err = this.compressionWriter.Write(data)
			} else {
				n1, err = this.writer.Write(data)
			}
			if n1 > 0 {
				this.sentBodyBytes += int64(n1)
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

			if this.bodyCopying {
				if this.compressionBodyWriter != nil {
					_, err := this.compressionBodyWriter.Write(data)
					if err != nil {
						remotelogs.Error("HTTP_WRITER", err.Error())
					}
				} else {
					this.body = append(this.body, data...)
				}
			}
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

// Send 发送响应
func (this *HTTPWriter) Send(status int, body string) {
	this.WriteHeader(status)
	_, _ = this.WriteString(body)
	this.isFinished = true
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
	if this.webpIsEncoding {
		defer func() {
			atomic.AddInt64(&webpTotalBufferSize, -this.size*32)
			webpBufferPool.Put(this.webpBuffer)
		}()
	}

	// webp writer
	if this.isOk && this.webpIsEncoding {
		var bufferLen = int64(this.webpBuffer.Len())
		atomic.AddInt64(&webpTotalBufferSize, bufferLen*4)

		// 需要把字节读取出来做备份，防止在image.Decode()过程中丢失
		var imageBytes = this.webpBuffer.Bytes()
		var imageData image.Image
		var gifImage *gif.GIF
		var isGif = strings.Contains(this.webpOriginContentType, "image/gif")

		var err error
		if this.webpOriginEncoding == "gzip" {
			this.Header().Del("Content-Encoding")
			var reader *gzip.Reader
			reader, err = gzip.NewReader(this.webpBuffer)
			if err == nil {
				defer func() {
					_ = reader.Close()
				}()
				if isGif {
					gifImage, err = gif.DecodeAll(reader)
				} else {
					imageData, _, err = image.Decode(reader)
				}
			}
		} else if this.webpOriginEncoding == "deflate" {
			this.Header().Del("Content-Encoding")
			var reader io.ReadCloser
			reader = flate.NewReader(this.webpBuffer)
			defer func() {
				_ = reader.Close()
			}()
			if isGif {
				gifImage, err = gif.DecodeAll(reader)
			} else {
				imageData, _, err = image.Decode(reader)
			}
		} else if this.webpOriginEncoding == "br" {
			this.Header().Del("Content-Encoding")
			var reader *brotli.Reader
			reader = brotli.NewReader(this.webpBuffer)
			if isGif {
				gifImage, err = gif.DecodeAll(reader)
			} else {
				imageData, _, err = image.Decode(reader)
			}
		} else {
			if isGif {
				gifImage, err = gif.DecodeAll(this.webpBuffer)
			} else {
				imageData, _, err = image.Decode(this.webpBuffer)
			}
		}
		if err != nil {
			this.Header().Set("Content-Type", this.webpOriginContentType)
			this.WriteHeader(http.StatusOK)
			_, _ = this.writer.Write(imageBytes)

			// 处理缓存
			if this.cacheWriter != nil {
				_ = this.cacheWriter.Discard()
			}
			this.cacheWriter = nil
		} else {
			var f = types.Float32(this.req.web.WebP.Quality)
			if f > 100 {
				f = 100
			}
			this.webpIsWriting = true

			if imageData != nil {
				err = gowebp.Encode(this, imageData, &gowebp.Options{
					Lossless: false,
					Quality:  f,
					Exact:    true,
				})
			} else if gifImage != nil {
				anim := gowebp.NewWebpAnimation(gifImage.Config.Width, gifImage.Config.Height, gifImage.LoopCount)
				anim.WebPAnimEncoderOptions.SetKmin(9)
				anim.WebPAnimEncoderOptions.SetKmax(17)
				defer anim.ReleaseMemory()
				webpConfig := gowebp.NewWebpConfig()
				//webpConfig.SetLossless(1)
				webpConfig.SetQuality(f)

				timeline := 0

				for i, img := range gifImage.Image {
					err = anim.AddFrame(img, timeline, webpConfig)
					if err != nil {
						break
					}
					timeline += gifImage.Delay[i] * 10
				}
				if err == nil {
					err = anim.AddFrame(nil, timeline, webpConfig)

					if err == nil {
						err = anim.Encode(this)
					}
				}
			}
			if err != nil {
				if !this.req.canIgnore(err) {
					remotelogs.Error("HTTP_WRITER", "encode webp failed: "+err.Error())
				}

				this.Header().Set("Content-Type", this.webpOriginContentType)
				this.WriteHeader(http.StatusOK)
				_, _ = this.writer.Write(imageBytes)

				// 处理缓存
				if this.cacheWriter != nil {
					_ = this.cacheWriter.Discard()
				}
				this.cacheWriter = nil
			}
		}

		atomic.AddInt64(&webpTotalBufferSize, -bufferLen*4)
		this.webpBuffer.Reset()
	}

	// compression writer
	if this.compressionWriter != nil {
		if this.bodyCopying && this.compressionBodyWriter != nil {
			_ = this.compressionBodyWriter.Close()
			this.body = this.compressionBodyBuffer.Bytes()
		}
		_ = this.compressionWriter.Close()
		this.compressionWriter = nil
	}

	// cache writer
	if this.cacheWriter != nil {
		if this.isOk {
			// 对比Content-Length
			contentLengthString := this.Header().Get("Content-Length")
			if len(contentLengthString) > 0 {
				contentLength := types.Int64(contentLengthString)
				if contentLength != this.cacheWriter.BodySize() {
					this.isOk = false
					_ = this.cacheWriter.Discard()
				}
			}

			if this.isOk {
				err := this.cacheWriter.Close()
				if err == nil {
					var expiredAt = this.cacheWriter.ExpiredAt()
					this.cacheStorage.AddToList(&caches.Item{
						Type:       this.cacheWriter.ItemType(),
						Key:        this.cacheWriter.Key(),
						ExpiredAt:  expiredAt,
						StaleAt:    expiredAt + int64(this.calculateStaleLife()),
						HeaderSize: this.cacheWriter.HeaderSize(),
						BodySize:   this.cacheWriter.BodySize(),
						Host:       this.req.ReqHost,
						ServerId:   this.req.ReqServer.Id,
					})
				}
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

// 准备Webp
func (this *HTTPWriter) prepareWebP(size int64) {
	if this.req.web != nil &&
		this.req.web.WebP != nil &&
		this.req.web.WebP.IsOn &&
		this.req.web.WebP.MatchResponse(this.Header().Get("Content-Type"), size, filepath.Ext(this.req.Path()), this.req.Format) &&
		this.req.web.WebP.MatchAccept(this.req.requestHeader("Accept")) &&
		atomic.LoadInt64(&webpTotalBufferSize) < webpMaxBufferSize {

		var contentEncoding = this.writer.Header().Get("Content-Encoding")
		switch contentEncoding {
		case "gzip", "deflate", "br":
			this.webpOriginEncoding = contentEncoding
		case "": // 空
		default:
			return
		}

		this.webpIsEncoding = true
		this.webpOriginContentType = this.Header().Get("Content-Type")
		this.webpBuffer = webpBufferPool.Get()

		this.Header().Del("Content-Length")
		this.Header().Set("Content-Type", "image/webp")

		atomic.AddInt64(&webpTotalBufferSize, size*32)
	}
}

// PrepareCompression 准备压缩
func (this *HTTPWriter) PrepareCompression(size int64) {
	if this.compressionConfig == nil || !this.compressionConfig.IsOn || this.compressionConfig.Level <= 0 {
		return
	}

	// 如果已经有编码则不处理
	var contentEncoding = this.writer.Header().Get("Content-Encoding")
	if len(contentEncoding) > 0 && (!this.compressionConfig.DecompressData || !lists.ContainsString([]string{"gzip", "deflate", "br"}, contentEncoding)) {
		return
	}

	// 尺寸和类型
	if !this.compressionConfig.MatchResponse(this.Header().Get("Content-Type"), size, filepath.Ext(this.req.Path()), this.req.Format) {
		return
	}

	// 判断Accept是否支持压缩
	compressionType, compressionEncoding, ok := this.compressionConfig.MatchAcceptEncoding(this.req.RawReq.Header.Get("Accept-Encoding"))
	if !ok {
		return
	}

	// 压缩前后如果编码一致，则不处理
	if compressionEncoding == contentEncoding {
		return
	}

	this.compressionType = compressionType

	// compression writer
	var err error = nil
	this.compressionWriter, err = compressions.NewWriter(this.writer, compressionType, int(this.compressionConfig.Level))
	if err != nil {
		remotelogs.Error("HTTP_WRITER", err.Error())
		return
	}

	// convert between encodings
	if len(contentEncoding) > 0 {
		this.compressionWriter, err = compressions.NewEncodingWriter(contentEncoding, this.compressionWriter)
		if err != nil {
			remotelogs.Error("HTTP_WRITER", err.Error())
			return
		}
	}

	// body copy
	if this.bodyCopying {
		this.compressionBodyBuffer = bytes.NewBuffer([]byte{})
		this.compressionBodyWriter, err = compressions.NewWriter(this.compressionBodyBuffer, compressionType, int(this.compressionConfig.Level))
		if err != nil {
			remotelogs.Error("HTTP_WRITER", err.Error())
		}
	}

	header := this.writer.Header()
	header.Set("Content-Encoding", compressionEncoding)
	header.Set("Vary", "Accept-Encoding")
	header.Del("Content-Length")
}

// 准备缓存
func (this *HTTPWriter) prepareCache(size int64) {
	if this.writer == nil {
		return
	}

	cachePolicy := this.req.ReqServer.HTTPCachePolicy
	if cachePolicy == nil || !cachePolicy.IsOn {
		return
	}

	cacheRef := this.req.cacheRef
	if cacheRef == nil || !cacheRef.IsOn {
		return
	}

	var addStatusHeader = this.req.web != nil && this.req.web.Cache != nil && this.req.web.Cache.AddStatusHeader

	// 不支持Range
	if len(this.Header().Get("Content-Range")) > 0 {
		this.req.varMapping["cache.status"] = "BYPASS"
		if addStatusHeader {
			this.Header().Set("X-Cache", "BYPASS, not supported Content-Range")
		}
		return
	}

	// 如果允许 ChunkedEncoding，就无需尺寸的判断，因为此时的 size 为 -1
	if !cacheRef.AllowChunkedEncoding && size < 0 {
		this.req.varMapping["cache.status"] = "BYPASS"
		if addStatusHeader {
			this.Header().Set("X-Cache", "BYPASS, ChunkedEncoding")
		}
		return
	}
	if size >= 0 && ((cacheRef.MaxSizeBytes() > 0 && size > cacheRef.MaxSizeBytes()) ||
		(cachePolicy.MaxSizeBytes() > 0 && size > cachePolicy.MaxSizeBytes()) || (cacheRef.MinSizeBytes() > size)) {
		this.req.varMapping["cache.status"] = "BYPASS"
		if addStatusHeader {
			this.Header().Set("X-Cache", "BYPASS, Content-Length")
		}
		return
	}

	// 检查状态
	if len(cacheRef.Status) > 0 && !lists.ContainsInt(cacheRef.Status, this.StatusCode()) {
		this.req.varMapping["cache.status"] = "BYPASS"
		if addStatusHeader {
			this.Header().Set("X-Cache", "BYPASS, Status: "+types.String(this.StatusCode()))
		}
		return
	}

	// Cache-Control
	if len(cacheRef.SkipResponseCacheControlValues) > 0 {
		cacheControl := this.writer.Header().Get("Cache-Control")
		if len(cacheControl) > 0 {
			values := strings.Split(cacheControl, ",")
			for _, value := range values {
				if cacheRef.ContainsCacheControl(strings.TrimSpace(value)) {
					this.req.varMapping["cache.status"] = "BYPASS"
					if addStatusHeader {
						this.Header().Set("X-Cache", "BYPASS, Cache-Control: "+cacheControl)
					}
					return
				}
			}
		}
	}

	// Set-Cookie
	if cacheRef.SkipResponseSetCookie && len(this.writer.Header().Get("Set-Cookie")) > 0 {
		this.req.varMapping["cache.status"] = "BYPASS"
		if addStatusHeader {
			this.Header().Set("X-Cache", "BYPASS, Set-Cookie")
		}
		return
	}

	// 校验其他条件
	if cacheRef.Conds != nil && cacheRef.Conds.HasResponseConds() && !cacheRef.Conds.MatchResponse(this.req.Format) {
		this.req.varMapping["cache.status"] = "BYPASS"
		if addStatusHeader {
			this.Header().Set("X-Cache", "BYPASS, ResponseConds")
		}
		return
	}

	// 打开缓存写入
	storage := caches.SharedManager.FindStorageWithPolicy(cachePolicy.Id)
	if storage == nil {
		this.req.varMapping["cache.status"] = "BYPASS"
		if addStatusHeader {
			this.Header().Set("X-Cache", "BYPASS, Storage")
		}
		return
	}

	this.req.varMapping["cache.status"] = "UPDATING"
	if addStatusHeader {
		this.Header().Set("X-Cache", "UPDATING")
	}

	this.cacheStorage = storage
	life := cacheRef.LifeSeconds()

	if life <= 0 {
		life = 60
	}

	// 支持源站设置的max-age
	if this.req.web.Cache != nil && this.req.web.Cache.EnableCacheControlMaxAge {
		var cacheControl = this.Header().Get("Cache-Control")
		var pieces = strings.Split(cacheControl, ";")
		for _, piece := range pieces {
			var eqIndex = strings.Index(piece, "=")
			if eqIndex > 0 && piece[:eqIndex] == "max-age" {
				var maxAge = types.Int64(piece[eqIndex+1:])
				if maxAge > 0 {
					life = maxAge
				}
			}
		}
	}

	expiredAt := utils.UnixTime() + life
	var cacheKey = this.req.cacheKey
	if this.webpIsEncoding {
		cacheKey += webpSuffix
	}
	cacheWriter, err := storage.OpenWriter(cacheKey, expiredAt, this.StatusCode())
	if err != nil {
		if !caches.CanIgnoreErr(err) {
			remotelogs.Error("HTTP_WRITER", "write cache failed: "+err.Error())
		}
		return
	}
	this.cacheWriter = cacheWriter

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

// 计算stale时长
func (this *HTTPWriter) calculateStaleLife() int {
	var staleLife = 600 // TODO 可以在缓存策略里设置此时间
	var staleConfig = this.req.web.Cache.Stale
	if staleConfig != nil && staleConfig.IsOn {
		// 从Header中读取stale-if-error
		var isDefinedInHeader = false
		if staleConfig.SupportStaleIfErrorHeader {
			var cacheControl = this.Header().Get("Cache-Control")
			var pieces = strings.Split(cacheControl, ",")
			for _, piece := range pieces {
				var eqIndex = strings.Index(piece, "=")
				if eqIndex > 0 && strings.TrimSpace(piece[:eqIndex]) == "stale-if-error" {
					// 这里预示着如果stale-if-error=0，可以关闭stale功能
					staleLife = types.Int(strings.TrimSpace(piece[eqIndex+1:]))
					isDefinedInHeader = true
					break
				}
			}
		}

		// 自定义
		if !isDefinedInHeader && staleConfig.Life != nil {
			staleLife = types.Int(staleConfig.Life.Duration().Seconds())
		}
	}
	return staleLife
}
