// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package nodes

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"github.com/TeaOSLab/EdgeCommon/pkg/nodeconfigs"
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs"
	"github.com/TeaOSLab/EdgeNode/internal/caches"
	"github.com/TeaOSLab/EdgeNode/internal/compressions"
	teaconst "github.com/TeaOSLab/EdgeNode/internal/const"
	"github.com/TeaOSLab/EdgeNode/internal/remotelogs"
	"github.com/TeaOSLab/EdgeNode/internal/utils"
	"github.com/TeaOSLab/EdgeNode/internal/utils/fasttime"
	"github.com/TeaOSLab/EdgeNode/internal/utils/readers"
	setutils "github.com/TeaOSLab/EdgeNode/internal/utils/sets"
	"github.com/TeaOSLab/EdgeNode/internal/utils/writers"
	_ "github.com/biessek/golang-ico"
	"github.com/iwind/TeaGo/lists"
	"github.com/iwind/TeaGo/types"
	"github.com/iwind/gowebp"
	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/webp"
	"image"
	"image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net"
	"net/http"
	"net/textproto"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
)

var webpMaxBufferSize int64 = 1_000_000_000
var webpTotalBufferSize int64 = 0
var webpIgnoreURLSet = setutils.NewFixedSet(131072)

func init() {
	if !teaconst.IsMain {
		return
	}

	var systemMemory = utils.SystemMemoryGB() / 8
	if systemMemory > 0 {
		webpMaxBufferSize = int64(systemMemory) << 30
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

	statusCode      int
	sentBodyBytes   int64
	sentHeaderBytes int64

	isOk       bool // 是否完全成功
	isFinished bool // 是否已完成

	// Partial
	isPartial        bool
	partialFileIsNew bool

	// WebP
	webpIsEncoding        bool
	webpOriginContentType string

	// Compression
	compressionConfig      *serverconfigs.HTTPCompressionConfig
	compressionCacheWriter caches.Writer

	// Cache
	cacheStorage    caches.StorageInterface
	cacheWriter     caches.Writer
	cacheIsFinished bool

	cacheReader       caches.Reader
	cacheReaderSuffix string
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
	// 清理以前数据，防止重试时发生异常错误
	if this.compressionCacheWriter != nil {
		_ = this.compressionCacheWriter.Discard()
		this.compressionCacheWriter = nil
	}

	if this.cacheWriter != nil {
		_ = this.cacheWriter.Discard()
		this.cacheWriter = nil
	}

	// 新的请求相关数据
	this.size = size
	this.statusCode = status

	// 是否为区间请求
	this.isPartial = status == http.StatusPartialContent

	// 不支持对GET以外的方法返回的Partial内容的缓存
	if this.isPartial && this.req.Method() != http.MethodGet {
		enableCache = false
	}

	if resp != nil && resp.Body != nil {
		cacheReader, ok := resp.Body.(caches.Reader)
		if ok {
			this.cacheReader = cacheReader
		}

		this.rawReader = resp.Body

		if enableCache {
			this.PrepareCache(resp, size)
		}
		if !this.isPartial {
			this.PrepareWebP(resp, size)
		}
		this.PrepareCompression(resp, size)
	}

	// 是否限速写入
	if this.req.web != nil &&
		this.req.web.RequestLimit != nil &&
		this.req.web.RequestLimit.IsOn &&
		this.req.web.RequestLimit.OutBandwidthPerConnBytes() > 0 {
		this.writer = writers.NewRateLimitWriter(this.req.RawReq.Context(), this.writer, this.req.web.RequestLimit.OutBandwidthPerConnBytes())
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
	if this.isPartial {
		if !cacheRef.AllowPartialContent {
			this.req.varMapping["cache.status"] = "BYPASS"
			if addStatusHeader {
				this.Header().Set("X-Cache", "BYPASS, not supported partial content")
			}
			return
		}
		if this.cacheStorage.Policy().Type != serverconfigs.CachePolicyStorageFile {
			this.req.varMapping["cache.status"] = "BYPASS"
			if addStatusHeader {
				this.Header().Set("X-Cache", "BYPASS, not supported partial content in memory storage")
			}
			return
		}
	}

	// 如果允许 ChunkedEncoding，就无需尺寸的判断，因为此时的 size 为 -1
	if !cacheRef.AllowChunkedEncoding && size < 0 {
		this.req.varMapping["cache.status"] = "BYPASS"
		if addStatusHeader {
			this.Header().Set("X-Cache", "BYPASS, ChunkedEncoding")
		}
		return
	}

	var contentSize = size
	if this.isPartial {
		// 从Content-Range中读取内容总长度
		var contentRange = this.Header().Get("Content-Range")
		_, totalSize := httpRequestParseContentRangeHeader(contentRange)
		if totalSize > 0 {
			contentSize = totalSize
		}
	}
	if contentSize >= 0 && ((cacheRef.MaxSizeBytes() > 0 && contentSize > cacheRef.MaxSizeBytes()) ||
		(cachePolicy.MaxSizeBytes() > 0 && contentSize > cachePolicy.MaxSizeBytes()) || (cacheRef.MinSizeBytes() > contentSize)) {
		this.req.varMapping["cache.status"] = "BYPASS"
		if addStatusHeader {
			this.Header().Set("X-Cache", "BYPASS, Content-Length")
		}
		return
	}

	// 检查状态
	if !cacheRef.MatchStatus(this.StatusCode()) {
		this.req.varMapping["cache.status"] = "BYPASS"
		if addStatusHeader {
			this.Header().Set("X-Cache", "BYPASS, Status: "+types.String(this.StatusCode()))
		}
		return
	}

	// Cache-Control
	if len(cacheRef.SkipResponseCacheControlValues) > 0 {
		var cacheControl = this.GetHeader("Cache-Control")
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
	if cacheRef.SkipResponseSetCookie && len(this.GetHeader("Set-Cookie")) > 0 {
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
	var life = cacheRef.LifeSeconds()

	if life <= 0 {
		life = 60
	}

	// 支持源站设置的max-age
	if this.req.web.Cache != nil && this.req.web.Cache.EnableCacheControlMaxAge {
		var cacheControl = this.GetHeader("Cache-Control")
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

	var expiresAt = fasttime.Now().Unix() + life

	if this.req.isLnRequest {
		// 返回上级节点过期时间
		this.SetHeader(LNExpiresHeader, []string{types.String(expiresAt)})
	} else {
		var expiresHeader = this.Header().Get(LNExpiresHeader)
		if len(expiresHeader) > 0 {
			this.Header().Del(LNExpiresHeader)

			var expiresHeaderInt64 = types.Int64(expiresHeader)
			if expiresHeaderInt64 > 0 {
				expiresAt = expiresHeaderInt64
			}
		}
	}

	var cacheKey = this.req.cacheKey
	if this.isPartial {
		cacheKey += caches.SuffixPartial
	}

	// 待写入尺寸
	var totalSize = size
	if this.isPartial {
		var contentRange = resp.Header.Get("Content-Range")
		if len(contentRange) > 0 {
			_, partialTotalSize := httpRequestParseContentRangeHeader(contentRange)
			if partialTotalSize > 0 && partialTotalSize > totalSize {
				totalSize = partialTotalSize
			}
		}
	}

	// 先清理以前的
	if this.cacheWriter != nil {
		_ = this.cacheWriter.Discard()
	}

	cacheWriter, err := storage.OpenWriter(cacheKey, expiresAt, this.StatusCode(), this.calculateHeaderLength(), totalSize, cacheRef.MaxSizeBytes(), this.isPartial)
	if err != nil {
		if err == caches.ErrEntityTooLarge && addStatusHeader {
			this.Header().Set("X-Cache", "BYPASS, entity too large")
		}

		if !caches.CanIgnoreErr(err) {
			remotelogs.Error("HTTP_WRITER", "write cache failed: "+err.Error())
			this.Header().Set("X-Cache", "BYPASS, write cache failed")
		} else {
			this.Header().Set("X-Cache", "BYPASS, "+err.Error())
		}
		return
	}
	this.cacheWriter = cacheWriter

	if this.isPartial {
		this.partialFileIsNew = cacheWriter.(*caches.PartialFileWriter).IsNew()
	}

	// 写入Header
	var headerBuf = utils.SharedBufferPool.Get()
	for k, v := range this.Header() {
		if this.shouldIgnoreHeader(k) {
			continue
		}
		for _, v1 := range v {
			if this.isPartial && k == "Content-Type" && strings.Contains(v1, "multipart/byteranges") {
				continue
			}
			_, err = headerBuf.Write([]byte(k + ":" + v1 + "\n"))
			if err != nil {
				utils.SharedBufferPool.Put(headerBuf)

				remotelogs.Error("HTTP_WRITER", "write cache failed: "+err.Error())
				_ = this.cacheWriter.Discard()
				this.cacheWriter = nil
				return
			}
		}
	}
	_, err = cacheWriter.WriteHeader(headerBuf.Bytes())
	utils.SharedBufferPool.Put(headerBuf)
	if err != nil {
		remotelogs.Error("HTTP_WRITER", "write cache failed: "+err.Error())
		_ = this.cacheWriter.Discard()
		this.cacheWriter = nil
		return
	}

	if this.isPartial {
		// content-range
		var contentRange = this.GetHeader("Content-Range")
		if len(contentRange) > 0 {
			start, total := httpRequestParseContentRangeHeader(contentRange)
			if start < 0 {
				return
			}
			if total > 0 {
				partialWriter, ok := cacheWriter.(*caches.PartialFileWriter)
				if !ok {
					return
				}
				partialWriter.SetBodyLength(total)
			}
			var filterReader = readers.NewFilterReaderCloser(resp.Body)
			this.cacheIsFinished = true
			var hasError = false
			filterReader.Add(func(p []byte, readErr error) error {
				// 这里不用处理readErr，因为只要把成功读取的部分写入缓存即可

				if hasError {
					return nil
				}

				var l = len(p)
				if l == 0 {
					return nil
				}
				defer func() {
					start += int64(l)
				}()
				err = cacheWriter.WriteAt(start, p)
				if err != nil {
					this.cacheIsFinished = false
					hasError = true
				}
				return nil
			})
			resp.Body = filterReader
			this.rawReader = filterReader
			return
		}

		// multipart/byteranges
		var contentType = this.GetHeader("Content-Type")
		if strings.Contains(contentType, "multipart/byteranges") {
			partialWriter, ok := cacheWriter.(*caches.PartialFileWriter)
			if !ok {
				return
			}

			var boundary = httpRequestParseBoundary(contentType)
			if len(boundary) == 0 {
				return
			}

			var reader = readers.NewByteRangesReaderCloser(resp.Body, boundary)
			var contentTypeWritten = false

			this.cacheIsFinished = true
			var hasError = false
			var writtenTotal = false
			reader.OnPartRead(func(start int64, end int64, total int64, data []byte, header textproto.MIMEHeader) {
				// TODO 如果 total 超出缓存限制，则不写入缓存数据，并且记录到某个内存表中，下次不再OpenWriter

				if hasError {
					return
				}

				// 写入total
				if !writtenTotal && total > 0 {
					partialWriter.SetBodyLength(total)
					writtenTotal = true
				}

				// 写入Content-Type
				if partialWriter.IsNew() && !contentTypeWritten {
					var realContentType = header.Get("Content-Type")
					if len(realContentType) > 0 {
						var h = []byte("Content-Type:" + realContentType + "\n")
						err = partialWriter.AppendHeader(h)
						if err != nil {
							hasError = true
							this.cacheIsFinished = false
							return
						}
					}

					contentTypeWritten = true
				}

				err := cacheWriter.WriteAt(start, data)
				if err != nil {
					hasError = true
					this.cacheIsFinished = false
				}
			})

			resp.Body = reader
			this.rawReader = reader
		}

		return
	}

	var cacheReader = readers.NewTeeReaderCloser(resp.Body, this.cacheWriter)
	resp.Body = cacheReader
	this.rawReader = cacheReader

	cacheReader.OnFail(func(err error) {
		if this.cacheWriter != nil {
			_ = this.cacheWriter.Discard()
		}
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

	// 集群配置
	var policy = this.req.nodeConfig.FindWebPImagePolicyWithClusterId(this.req.ReqServer.ClusterId)
	if policy == nil {
		policy = nodeconfigs.DefaultWebPImagePolicy
	}
	if !policy.IsOn {
		return
	}

	// 只有在开启了缓存之后，才会转换，防止占用的系统资源过高
	if policy.RequireCache && this.req.cacheRef == nil {
		return
	}

	// 限制最小和最大尺寸
	// TODO 需要将reader修改为LimitReader
	if resp.ContentLength == 0 {
		return
	}

	if resp.ContentLength > 0 && (resp.ContentLength < policy.MinLengthBytes() || (policy.MaxLengthBytes() > 0 && resp.ContentLength > policy.MaxLengthBytes())) {
		return
	}

	var contentType = this.GetHeader("Content-Type")

	if this.req.web != nil &&
		this.req.web.WebP != nil &&
		this.req.web.WebP.IsOn &&
		this.req.web.WebP.MatchResponse(contentType, size, filepath.Ext(this.req.Path()), this.req.Format) &&
		this.req.web.WebP.MatchAccept(this.req.requestHeader("Accept")) {
		// 检查是否已经因为尺寸过大而忽略
		if webpIgnoreURLSet.Has(this.req.URL()) {
			return
		}

		// 如果已经是WebP不再重复处理
		// TODO 考虑是否需要很严格的匹配
		if strings.Contains(contentType, "image/webp") {
			return
		}

		// 检查内存
		if atomic.LoadInt64(&webpTotalBufferSize) >= webpMaxBufferSize {
			return
		}

		var contentEncoding = this.GetHeader("Content-Encoding")
		switch contentEncoding {
		case "gzip", "deflate", "br", "zstd":
			reader, err := compressions.NewReader(resp.Body, contentEncoding)
			if err != nil {
				return
			}
			this.Header().Del("Content-Encoding")
			this.Header().Del("Content-Length")
			this.rawReader = reader
		case "": // 空
		default:
			return
		}

		this.webpOriginContentType = contentType
		this.webpIsEncoding = true
		resp.Body = io.NopCloser(&bytes.Buffer{})
		this.delayRead = true

		this.Header().Del("Content-Length")
		this.Header().Set("Content-Type", "image/webp")
	}
}

// PrepareCompression 准备压缩
func (this *HTTPWriter) PrepareCompression(resp *http.Response, size int64) {
	var method = this.req.Method()
	if method == http.MethodHead {
		return
	}

	if this.StatusCode() == http.StatusNoContent {
		return
	}

	var acceptEncodings = this.req.RawReq.Header.Get("Accept-Encoding")
	var contentEncoding = this.GetHeader("Content-Encoding")

	if this.compressionConfig == nil || !this.compressionConfig.IsOn {
		if lists.ContainsString([]string{"gzip", "deflate", "br", "zstd"}, contentEncoding) && !httpAcceptEncoding(acceptEncodings, contentEncoding) {
			reader, err := compressions.NewReader(resp.Body, contentEncoding)
			if err != nil {
				return
			}
			this.Header().Del("Content-Encoding")
			this.Header().Del("Content-Length")
			resp.Body = reader
		}
		return
	}

	// 分区内容不压缩，防止读取失败
	if !this.compressionConfig.EnablePartialContent && this.StatusCode() == http.StatusPartialContent {
		return
	}

	if this.compressionConfig.Level <= 0 {
		return
	}

	// 如果已经有编码则不处理
	if len(contentEncoding) > 0 && (!this.compressionConfig.DecompressData || !lists.ContainsString([]string{"gzip", "deflate", "br", "zstd"}, contentEncoding)) {
		return
	}

	// 尺寸和类型
	var contentType = this.GetHeader("Content-Type")
	if !this.compressionConfig.MatchResponse(contentType, size, filepath.Ext(this.req.Path()), this.req.Format) {
		return
	}

	// 判断Accept是否支持压缩
	compressionType, compressionEncoding, ok := this.compressionConfig.MatchAcceptEncoding(acceptEncodings)
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
		this.Header().Del("Content-Encoding")
		this.Header().Del("Content-Length")
		resp.Body = reader
	}

	// 需要放在compression cache writer之前
	var header = this.rawWriter.Header()
	header.Set("Content-Encoding", compressionEncoding)
	header.Set("Vary", "Accept-Encoding")
	header.Del("Content-Length")

	// compression cache writer
	// 只有在本身内容已经缓存的情况下才会写入缓存，防止同时写入缓存导致IO负载升高
	var cacheRef = this.req.cacheRef
	if !this.isPartial &&
		this.cacheStorage != nil &&
		cacheRef != nil &&
		(this.cacheReader != nil || (this.cacheStorage.Policy().SyncCompressionCache && this.cacheWriter != nil)) &&
		!this.webpIsEncoding {
		var cacheKey = ""
		var expiredAt int64 = 0

		if this.cacheReader != nil {
			cacheKey = this.req.cacheKey
			expiredAt = this.cacheReader.ExpiresAt()
		} else if this.cacheWriter != nil {
			cacheKey = this.cacheWriter.Key()
			expiredAt = this.cacheWriter.ExpiredAt()
		}

		if len(this.cacheReaderSuffix) > 0 {
			cacheKey += this.cacheReaderSuffix
		}

		compressionCacheWriter, err := this.cacheStorage.OpenWriter(cacheKey+caches.SuffixCompression+compressionEncoding, expiredAt, this.StatusCode(), this.calculateHeaderLength(), -1, cacheRef.MaxSizeBytes(), false)
		if err != nil {
			return
		}

		// 写入Header
		var headerBuffer = utils.SharedBufferPool.Get()
		for k, v := range this.Header() {
			if this.shouldIgnoreHeader(k) {
				continue
			}
			for _, v1 := range v {
				_, err = headerBuffer.Write([]byte(k + ":" + v1 + "\n"))
				if err != nil {
					utils.SharedBufferPool.Put(headerBuffer)
					remotelogs.Error("HTTP_WRITER", "write compression cache failed: "+err.Error())
					_ = compressionCacheWriter.Discard()
					compressionCacheWriter = nil
					return
				}
			}
		}

		_, err = compressionCacheWriter.WriteHeader(headerBuffer.Bytes())
		utils.SharedBufferPool.Put(headerBuffer)
		if err != nil {
			remotelogs.Error("HTTP_WRITER", "write compression cache failed: "+err.Error())
			_ = compressionCacheWriter.Discard()
			compressionCacheWriter = nil
			return
		}

		if compressionCacheWriter != nil {
			if this.compressionCacheWriter != nil {
				_ = this.compressionCacheWriter.Close()
			}
			this.compressionCacheWriter = compressionCacheWriter
			var teeWriter = writers.NewTeeWriterCloser(this.writer, compressionCacheWriter)
			teeWriter.OnFail(func(err error) {
				_ = compressionCacheWriter.Discard()
				this.compressionCacheWriter = nil
			})
			this.writer = teeWriter
		}
	}

	// compression writer
	compressionWriter, err := compressions.NewWriter(this.writer, compressionType, int(this.compressionConfig.Level))
	if err != nil {
		remotelogs.Error("HTTP_WRITER", err.Error())
		header.Del("Content-Encoding")
		if this.compressionCacheWriter != nil {
			_ = this.compressionCacheWriter.Discard()
		}
		return
	}
	this.writer = compressionWriter
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

// GetHeader 读取Header值
func (this *HTTPWriter) GetHeader(name string) string {
	return this.Header().Get(name)
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
	var newHeaders = this.rawWriter.Header()
	for key, value := range header {
		if key == "Connection" {
			continue
		}
		switch key {
		case "Accept-CH", "ETag", "Content-MD5", "IM", "P3P", "WWW-Authenticate", "X-Request-ID":
			newHeaders[key] = value
		default:
			newHeaders[http.CanonicalHeaderKey(key)] = value
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

// SentHeaderBytes 计算发送的Header字节数
func (this *HTTPWriter) SentHeaderBytes() int64 {
	if this.sentHeaderBytes > 0 {
		return this.sentHeaderBytes
	}
	for k, v := range this.Header() {
		for _, v1 := range v {
			this.sentHeaderBytes += int64(len(k) + 2 + len(v1) + 1)
		}
	}
	return this.sentHeaderBytes
}

func (this *HTTPWriter) SetSentHeaderBytes(sentHeaderBytes int64) {
	this.sentHeaderBytes = sentHeaderBytes
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
	this.req.ProcessResponseHeaders(this.Header(), status)

	// content-length
	_, hasContentLength := this.Header()["Content-Length"]
	if !hasContentLength {
		this.Header()["Content-Length"] = []string{types.String(len(body))}
	}

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
		return 0, fmt.Errorf("open file '%s' failed: %w", path, err)
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

// SendResp 发送响应对象
func (this *HTTPWriter) SendResp(resp *http.Response) (int64, error) {
	this.isFinished = true

	for k, v := range resp.Header {
		this.SetHeader(k, v)
	}

	this.WriteHeader(resp.StatusCode)
	var bufPool = this.req.bytePool(resp.ContentLength)
	var buf = bufPool.Get()
	defer bufPool.Put(buf)

	return io.CopyBuffer(this, resp.Body, buf)
}

// Redirect 跳转
func (this *HTTPWriter) Redirect(status int, url string) {
	httpRedirect(this, this.req.RawReq, url, status)
	this.isFinished = true
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

	var resp = &http.Response{}
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
	this.finishWebP()
	this.finishRequest()
	this.finishCache()
	this.finishCompression()

	// 统计
	if this.sentBodyBytes == 0 {
		this.sentBodyBytes = this.counterWriter.TotalBytes()
	}
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
	var staleLife = caches.DefaultStaleCacheSeconds
	var staleConfig = this.req.web.Cache.Stale
	if staleConfig != nil && staleConfig.IsOn {
		// 从Header中读取stale-if-error
		var isDefinedInHeader = false
		if staleConfig.SupportStaleIfErrorHeader {
			var cacheControl = this.GetHeader("Cache-Control")
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

// 结束WebP
func (this *HTTPWriter) finishWebP() {
	// 处理WebP
	if this.webpIsEncoding {
		var webpCacheWriter caches.Writer

		// 准备WebP Cache
		if this.cacheReader != nil || this.cacheWriter != nil {
			var cacheKey = ""
			var expiredAt int64 = 0

			if this.cacheReader != nil {
				cacheKey = this.req.cacheKey + caches.SuffixWebP
				expiredAt = this.cacheReader.ExpiresAt()
			} else if this.cacheWriter != nil {
				cacheKey = this.cacheWriter.Key() + caches.SuffixWebP
				expiredAt = this.cacheWriter.ExpiredAt()
			}

			webpCacheWriter, _ = this.cacheStorage.OpenWriter(cacheKey, expiredAt, this.StatusCode(), -1, -1, -1, false)
			if webpCacheWriter != nil {
				// 写入Header
				for k, v := range this.Header() {
					if this.shouldIgnoreHeader(k) {
						continue
					}

					// 这里是原始的数据，不需要内容编码
					if k == "Content-Encoding" || k == "Transfer-Encoding" {
						continue
					}
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
						if webpCacheWriter != nil {
							_ = webpCacheWriter.Discard()
						}
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
			if gifImage != nil && (gifImage.Config.Width > gowebp.WebPMaxDimension || gifImage.Config.Height > gowebp.WebPMaxDimension) {
				webpIgnoreURLSet.Push(this.req.URL())
				return
			}
		} else {
			imageData, _, err = image.Decode(reader)
			if imageData != nil {
				var bound = imageData.Bounds()
				if bound.Max.X > gowebp.WebPMaxDimension || bound.Max.Y > gowebp.WebPMaxDimension {
					webpIgnoreURLSet.Push(this.req.URL())
					return
				}
			}
		}

		if err != nil {
			// 发生了错误终止处理
			webpIgnoreURLSet.Push(this.req.URL())
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
			var anim = gowebp.NewWebpAnimation(gifImage.Config.Width, gifImage.Config.Height, gifImage.LoopCount)

			anim.WebPAnimEncoderOptions.SetKmin(9)
			anim.WebPAnimEncoderOptions.SetKmax(17)
			var webpConfig = gowebp.NewWebpConfig()
			//webpConfig.SetLossless(1)
			webpConfig.SetQuality(f)

			var timeline = 0
			var lastErr error
			for i, img := range gifImage.Image {
				err = anim.AddFrame(img, timeline, webpConfig)
				if err != nil {
					// 有错误直接跳过
					lastErr = err
					err = nil
				}
				timeline += gifImage.Delay[i] * 10
			}
			if lastErr != nil {
				remotelogs.Error("HTTP_WRITER", "'"+this.req.URL()+"' encode webp failed: "+lastErr.Error())
			}
			err = anim.AddFrame(nil, timeline, webpConfig)

			if err == nil {
				err = anim.Encode(this.writer)
			}

			anim.ReleaseMemory()
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
}

// 结束缓存相关处理
func (this *HTTPWriter) finishCache() {
	// 缓存
	if this.cacheWriter != nil {
		if this.isOk && this.cacheIsFinished {
			// 对比缓存前后的Content-Length
			var method = this.req.Method()
			if method != http.MethodHead && this.StatusCode() != http.StatusNoContent && !this.isPartial {
				var contentLengthString = this.GetHeader("Content-Length")
				if len(contentLengthString) > 0 {
					var contentLength = types.Int64(contentLengthString)
					if contentLength != this.cacheWriter.BodySize() {
						this.isOk = false
						_ = this.cacheWriter.Discard()
						this.cacheWriter = nil
					}
				}
			}

			if this.isOk && this.cacheWriter != nil {
				err := this.cacheWriter.Close()
				if err == nil {
					if !this.isPartial || this.partialFileIsNew {
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
			}
		} else {
			if !this.isPartial || !this.cacheIsFinished {
				_ = this.cacheWriter.Discard()
			} else {
				// Partial的文件内容不删除
				err := this.cacheWriter.Close()
				if err == nil && this.partialFileIsNew {
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
		}
	}
}

// 结束压缩相关处理
func (this *HTTPWriter) finishCompression() {
	if this.compressionCacheWriter != nil {
		if this.isOk {
			err := this.compressionCacheWriter.Close()
			if err == nil {
				var expiredAt = this.compressionCacheWriter.ExpiredAt()
				this.cacheStorage.AddToList(&caches.Item{
					Type:       this.compressionCacheWriter.ItemType(),
					Key:        this.compressionCacheWriter.Key(),
					ExpiredAt:  expiredAt,
					StaleAt:    expiredAt + int64(this.calculateStaleLife()),
					HeaderSize: this.compressionCacheWriter.HeaderSize(),
					BodySize:   this.compressionCacheWriter.BodySize(),
					Host:       this.req.ReqHost,
					ServerId:   this.req.ReqServer.Id,
				})
			}
		} else {
			_ = this.compressionCacheWriter.Discard()
		}
	}
}

// 最终关闭
func (this *HTTPWriter) finishRequest() {
	if this.writer != nil {
		_ = this.writer.Close()
	}

	if this.rawReader != nil {
		_ = this.rawReader.Close()
	}
}

// 计算Header长度
func (this *HTTPWriter) calculateHeaderLength() (result int) {
	for k, v := range this.Header() {
		if this.shouldIgnoreHeader(k) {
			continue
		}
		for _, v1 := range v {
			result += len(k) + 1 /**:**/ + len(v1) + 1 /**\n**/
		}
	}
	return
}

func (this *HTTPWriter) shouldIgnoreHeader(name string) bool {
	switch name {
	case "Set-Cookie", "Strict-Transport-Security", "Alt-Svc", "Upgrade", "X-Cache":
		return true
	default:
		return (this.isPartial && name == "Content-Range")
	}
}
