// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package nodes

import (
	"bufio"
	"bytes"
	"errors"
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs"
	"github.com/TeaOSLab/EdgeNode/internal/caches"
	"github.com/TeaOSLab/EdgeNode/internal/compressions"
	"github.com/TeaOSLab/EdgeNode/internal/remotelogs"
	"github.com/TeaOSLab/EdgeNode/internal/utils"
	"github.com/TeaOSLab/EdgeNode/internal/utils/readers"
	"github.com/TeaOSLab/EdgeNode/internal/utils/writers"
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
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
)

// webp相关配置
const webpSuffix = "@GOEDGE_WEBP"

var webpMaxBufferSize int64 = 1_000_000_000
var webpTotalBufferSize int64 = 0

func init() {
	var systemMemory = utils.SystemMemoryGB() / 8
	if systemMemory > 0 {
		webpMaxBufferSize = int64(systemMemory) * 1024 * 1024 * 1024
	}
}

// HTTPWriter 响应Writer
type HTTPWriter struct {
	req       *HTTPRequest
	rawWriter http.ResponseWriter

	rawReader io.ReadCloser
	delayRead bool

	counterWriter *writers.BytesCounterWriter
	writer        io.WriteCloser

	size int64

	statusCode    int
	sentBodyBytes int64

	isOk       bool // 是否完全成功
	isFinished bool // 是否已完成

	// WebP
	webpIsEncoding        bool
	webpOriginContentType string

	// Compression
	compressionConfig *serverconfigs.HTTPCompressionConfig

	// Cache
	cacheStorage    caches.StorageInterface
	cacheWriter     caches.Writer
	cacheIsFinished bool
}

// NewHTTPWriter 包装对象
func NewHTTPWriter(req *HTTPRequest, httpResponseWriter http.ResponseWriter) *HTTPWriter {
	var counterWriter = writers.NewBytesCounterWriter(httpResponseWriter)
	return &HTTPWriter{
		req:           req,
		rawWriter:     httpResponseWriter,
		writer:        counterWriter,
		counterWriter: counterWriter,
	}
}

// Prepare 准备输出
func (this *HTTPWriter) Prepare(resp *http.Response, size int64, status int, enableCache bool) (delayHeaders bool) {
	this.size = size
	this.statusCode = status

	if resp != nil {
		this.rawReader = resp.Body

		if enableCache {
			this.PrepareCache(resp, size)
		}
		this.PrepareWebP(resp, size)
		this.PrepareCompression(resp, size)
	}

	// 是否限速写入
	if this.req.web != nil &&
		this.req.web.RequestLimit != nil &&
		this.req.web.RequestLimit.IsOn &&
		this.req.web.RequestLimit.OutBandwidthPerConnBytes() > 0 {
		this.writer = writers.NewRateLimitWriter(this.writer, this.req.web.RequestLimit.OutBandwidthPerConnBytes())
	}

	return
}

// PrepareCache 准备缓存
func (this *HTTPWriter) PrepareCache(resp *http.Response, size int64) {
	if resp == nil {
		return
	}

	var cachePolicy = this.req.ReqServer.HTTPCachePolicy
	if cachePolicy == nil || !cachePolicy.IsOn {
		return
	}

	var cacheRef = this.req.cacheRef
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
		var cacheControl = this.Header().Get("Cache-Control")
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
	if cacheRef.SkipResponseSetCookie && len(this.Header().Get("Set-Cookie")) > 0 {
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
	var storage = caches.SharedManager.FindStorageWithPolicy(cachePolicy.Id)
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

	var expiredAt = utils.UnixTime() + life
	var cacheKey = this.req.cacheKey
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

	var cacheReader = readers.NewTeeReaderCloser(resp.Body, this.cacheWriter)
	resp.Body = cacheReader
	this.rawReader = cacheReader

	cacheReader.OnFail(func(err error) {
		_ = this.cacheWriter.Discard()
		this.cacheWriter = nil
	})
	cacheReader.OnEOF(func() {
		this.cacheIsFinished = true
	})
}

// PrepareWebP 准备WebP
func (this *HTTPWriter) PrepareWebP(resp *http.Response, size int64) {
	if resp == nil {
		return
	}

	var contentType = this.Header().Get("Content-Type")

	if this.req.web != nil &&
		this.req.web.WebP != nil &&
		this.req.web.WebP.IsOn &&
		this.req.web.WebP.MatchResponse(contentType, size, filepath.Ext(this.req.Path()), this.req.Format) &&
		this.req.web.WebP.MatchAccept(this.req.requestHeader("Accept")) {
		// 如果已经是WebP不再重复处理
		// TODO 考虑是否需要很严格的匹配
		if strings.Contains(contentType, "image/webp") {
			return
		}

		// 检查内存
		if atomic.LoadInt64(&webpTotalBufferSize) >= webpMaxBufferSize {
			return
		}

		var contentEncoding = resp.Header.Get("Content-Encoding")
		switch contentEncoding {
		case "gzip", "deflate", "br":
			reader, err := compressions.NewReader(resp.Body, contentEncoding)
			if err != nil {
				return
			}
			this.Header().Del("Content-Encoding")
			this.rawReader = reader
		case "": // 空
		default:
			return
		}

		this.webpOriginContentType = contentType
		this.webpIsEncoding = true
		resp.Body = ioutil.NopCloser(&bytes.Buffer{})
		this.delayRead = true

		this.Header().Del("Content-Length")
		this.Header().Set("Content-Type", "image/webp")
	}
}

// PrepareCompression 准备压缩
func (this *HTTPWriter) PrepareCompression(resp *http.Response, size int64) {
	if this.compressionConfig == nil || !this.compressionConfig.IsOn || this.compressionConfig.Level <= 0 {
		return
	}

	// 如果已经有编码则不处理
	var contentEncoding = this.rawWriter.Header().Get("Content-Encoding")
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

	if len(contentEncoding) > 0 && resp != nil {
		if !this.compressionConfig.DecompressData {
			return
		}

		reader, err := compressions.NewReader(resp.Body, contentEncoding)
		if err != nil {
			return
		}
		resp.Body = reader
	}

	// compression writer
	var err error = nil
	compressionWriter, err := compressions.NewWriter(this.writer, compressionType, int(this.compressionConfig.Level))
	if err != nil {
		remotelogs.Error("HTTP_WRITER", err.Error())
		return
	}
	this.writer = compressionWriter

	header := this.rawWriter.Header()
	header.Set("Content-Encoding", compressionEncoding)
	header.Set("Vary", "Accept-Encoding")
	header.Del("Content-Length")
}

// SetCompression 设置内容压缩配置
func (this *HTTPWriter) SetCompression(config *serverconfigs.HTTPCompressionConfig) {
	this.compressionConfig = config
}

// Raw 包装前的原始的Writer
func (this *HTTPWriter) Raw() http.ResponseWriter {
	return this.rawWriter
}

// Header 获取Header
func (this *HTTPWriter) Header() http.Header {
	if this.rawWriter == nil {
		return http.Header{}
	}
	return this.rawWriter.Header()
}

// DeleteHeader 删除Header
func (this *HTTPWriter) DeleteHeader(name string) {
	this.rawWriter.Header().Del(name)
}

// SetHeader 设置Header
func (this *HTTPWriter) SetHeader(name string, values []string) {
	this.rawWriter.Header()[name] = values
}

// AddHeaders 添加一组Header
func (this *HTTPWriter) AddHeaders(header http.Header) {
	if this.rawWriter == nil {
		return
	}
	for key, value := range header {
		if key == "Connection" {
			continue
		}
		for _, v := range value {
			this.rawWriter.Header().Add(key, v)
		}
	}
}

// Write 写入数据
func (this *HTTPWriter) Write(data []byte) (n int, err error) {
	if this.webpIsEncoding {
		return
	}
	n, err = this.writer.Write(data)

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
	if this.rawWriter != nil {
		this.rawWriter.WriteHeader(statusCode)
	}
	this.statusCode = statusCode
}

// Send 直接发送内容，并终止请求
func (this *HTTPWriter) Send(status int, body string) {
	this.WriteHeader(status)
	_, _ = this.WriteString(body)
	this.isFinished = true
}

// SendFile 发送文件内容，并终止请求
func (this *HTTPWriter) SendFile(status int, path string) (int64, error) {
	this.WriteHeader(status)
	this.isFinished = true

	fp, err := os.OpenFile(path, os.O_RDONLY, 0444)
	if err != nil {
		return 0, errors.New("open file '" + path + "' failed: " + err.Error())
	}
	defer func() {
		_ = fp.Close()
	}()

	stat, err := fp.Stat()
	if err != nil {
		return 0, err
	}
	if stat.IsDir() {
		return 0, errors.New("open file '" + path + "' failed: it is a directory")
	}

	var bufPool = this.req.bytePool(stat.Size())
	var buf = bufPool.Get()
	defer bufPool.Put(buf)

	written, err := io.CopyBuffer(this, fp, buf)
	if err != nil {
		return written, err
	}

	return written, nil
}

// StatusCode 读取状态码
func (this *HTTPWriter) StatusCode() int {
	if this.statusCode == 0 {
		return http.StatusOK
	}
	return this.statusCode
}

// HeaderData 读取Header二进制数据
func (this *HTTPWriter) HeaderData() []byte {
	if this.rawWriter == nil {
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
	// 处理WebP
	if this.webpIsEncoding {
		var webpCacheWriter caches.Writer

		// 准备WebP Cache
		if this.cacheWriter != nil {
			var cacheKey = this.cacheWriter.Key() + webpSuffix

			webpCacheWriter, _ = this.cacheStorage.OpenWriter(cacheKey, this.cacheWriter.ExpiredAt(), this.StatusCode())
			if webpCacheWriter != nil {
				// 写入Header
				for k, v := range this.Header() {
					for _, v1 := range v {
						_, err := webpCacheWriter.WriteHeader([]byte(k + ":" + v1 + "\n"))
						if err != nil {
							remotelogs.Error("HTTP_WRITER", "write webp cache failed: "+err.Error())
							_ = webpCacheWriter.Discard()
							webpCacheWriter = nil
							break
						}
					}
				}

				if webpCacheWriter != nil {
					var teeWriter = writers.NewTeeWriterCloser(this.writer, webpCacheWriter)
					teeWriter.OnFail(func(err error) {
						_ = webpCacheWriter.Discard()
						webpCacheWriter = nil
					})
					this.writer = teeWriter
				}
			}
		}

		var reader = readers.NewBytesCounterReader(this.rawReader)

		var imageData image.Image
		var gifImage *gif.GIF
		var isGif = strings.Contains(this.webpOriginContentType, "image/gif")
		var err error
		if isGif {
			gifImage, err = gif.DecodeAll(reader)
		} else {
			imageData, _, err = image.Decode(reader)
		}

		if err != nil {
			return
		}

		var totalBytes = reader.TotalBytes()
		atomic.AddInt64(&webpTotalBufferSize, totalBytes)
		defer func() {
			atomic.AddInt64(&webpTotalBufferSize, -totalBytes)
		}()

		var f = types.Float32(this.req.web.WebP.Quality)
		if f > 100 {
			f = 100
		}

		if imageData != nil {
			err = gowebp.Encode(this.writer, imageData, &gowebp.Options{
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
					err = anim.Encode(this.writer)
				}
			}
		}

		if err != nil && !this.req.canIgnore(err) {
			remotelogs.Error("HTTP_WRITER", "'"+this.req.URL()+"' encode webp failed: "+err.Error())
		}

		if err == nil && webpCacheWriter != nil {
			err = webpCacheWriter.Close()
			if err != nil {
				_ = webpCacheWriter.Discard()
			} else {
				this.cacheStorage.AddToList(&caches.Item{
					Type:       webpCacheWriter.ItemType(),
					Key:        webpCacheWriter.Key(),
					ExpiredAt:  webpCacheWriter.ExpiredAt(),
					StaleAt:    webpCacheWriter.ExpiredAt() + int64(this.calculateStaleLife()),
					HeaderSize: webpCacheWriter.HeaderSize(),
					BodySize:   webpCacheWriter.BodySize(),
					Host:       this.req.ReqHost,
					ServerId:   this.req.ReqServer.Id,
				})
			}
		}
	}

	if this.writer != nil {
		_ = this.writer.Close()
	}

	if this.rawReader != nil {
		_ = this.rawReader.Close()
	}

	// 缓存
	if this.cacheWriter != nil {
		if this.isOk && this.cacheIsFinished {
			// 对比Content-Length
			var contentLengthString = this.Header().Get("Content-Length")
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

	this.sentBodyBytes = this.counterWriter.TotalBytes()
}

// Hijack Hijack
func (this *HTTPWriter) Hijack() (conn net.Conn, buf *bufio.ReadWriter, err error) {
	hijack, ok := this.rawWriter.(http.Hijacker)
	if ok {
		return hijack.Hijack()
	}
	return
}

// Flush Flush
func (this *HTTPWriter) Flush() {
	flusher, ok := this.rawWriter.(http.Flusher)
	if ok {
		flusher.Flush()
	}
}

// DelayRead 是否延迟读取Reader
func (this *HTTPWriter) DelayRead() bool {
	return this.delayRead
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
