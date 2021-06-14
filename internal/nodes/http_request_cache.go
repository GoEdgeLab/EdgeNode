package nodes

import (
	"bytes"
	"errors"
	"github.com/TeaOSLab/EdgeNode/internal/caches"
	"github.com/TeaOSLab/EdgeNode/internal/remotelogs"
	"github.com/iwind/TeaGo/logs"
	"net/http"
	"strconv"
	"time"
)

// 读取缓存
func (this *HTTPRequest) doCacheRead() (shouldStop bool) {
	cachePolicy := sharedNodeConfig.HTTPCachePolicy
	if cachePolicy == nil || !cachePolicy.IsOn {
		return
	}

	if this.web.Cache == nil || !this.web.Cache.IsOn || (len(cachePolicy.CacheRefs) == 0 && len(this.web.Cache.CacheRefs) == 0) {
		return
	}
	var addStatusHeader = this.web.Cache.AddStatusHeader
	if addStatusHeader {
		defer func() {
			cacheStatus := this.varMapping["cache.status"]
			if cacheStatus != "HIT" {
				this.writer.Header().Set("X-Cache", cacheStatus)
			}
		}()
	}

	// 检查服务独立的缓存条件
	refType := ""
	for _, cacheRef := range this.web.Cache.CacheRefs {
		if !cacheRef.IsOn ||
			cacheRef.Conds == nil ||
			!cacheRef.Conds.HasRequestConds() {
			continue
		}
		if cacheRef.Conds.MatchRequest(this.Format) {
			if cacheRef.IsReverse {
				return
			}
			this.cacheRef = cacheRef
			refType = "server"
			break
		}
	}
	if this.cacheRef == nil {
		// 检查策略默认的缓存条件
		for _, cacheRef := range cachePolicy.CacheRefs {
			if !cacheRef.IsOn ||
				cacheRef.Conds == nil ||
				!cacheRef.Conds.HasRequestConds() {
				continue
			}
			if cacheRef.Conds.MatchRequest(this.Format) {
				if cacheRef.IsReverse {
					return
				}
				this.cacheRef = cacheRef
				refType = "policy"
				break
			}
		}

		if this.cacheRef == nil {
			return
		}
	}

	// 相关变量
	this.varMapping["cache.policy.name"] = cachePolicy.Name
	this.varMapping["cache.policy.id"] = strconv.FormatInt(cachePolicy.Id, 10)
	this.varMapping["cache.policy.type"] = cachePolicy.Type

	// Cache-Pragma
	if this.cacheRef.EnableRequestCachePragma {
		if this.RawReq.Header.Get("Cache-Control") == "no-cache" || this.RawReq.Header.Get("Pragma") == "no-cache" {
			this.cacheRef = nil
			return
		}
	}

	// TODO 支持Vary Header

	// 检查是否有缓存
	key := this.Format(this.cacheRef.Key)
	if len(key) == 0 {
		this.cacheRef = nil
		return
	}
	this.cacheKey = key

	// 读取缓存
	storage := caches.SharedManager.FindStorageWithPolicy(cachePolicy.Id)
	if storage == nil {
		this.cacheRef = nil
		return
	}

	buf := bytePool32k.Get()
	defer func() {
		bytePool32k.Put(buf)
	}()

	reader, err := storage.OpenReader(key)
	if err != nil {
		if err == caches.ErrNotFound {
			// cache相关变量
			this.varMapping["cache.status"] = "MISS"
			return
		}

		if !this.canIgnore(err) {
			remotelogs.Warn("HTTP_REQUEST_CACHE", "read from cache failed: "+err.Error())
		}
		return
	}
	defer func() {
		_ = reader.Close()
	}()

	this.varMapping["cache.status"] = "HIT"
	this.logAttrs["cache.status"] = "HIT"

	// 读取Header
	headerBuf := []byte{}
	err = reader.ReadHeader(buf, func(n int) (goNext bool, err error) {
		headerBuf = append(headerBuf, buf[:n]...)
		for {
			nIndex := bytes.Index(headerBuf, []byte{'\n'})
			if nIndex >= 0 {
				row := headerBuf[:nIndex]
				spaceIndex := bytes.Index(row, []byte{':'})
				if spaceIndex <= 0 {
					return false, errors.New("invalid header '" + string(row) + "'")
				}

				this.writer.Header().Set(string(row[:spaceIndex]), string(row[spaceIndex+1:]))
				headerBuf = headerBuf[nIndex+1:]
			} else {
				break
			}
		}
		return true, nil
	})
	if err != nil {
		if !this.canIgnore(err) {
			remotelogs.Warn("HTTP_REQUEST_CACHE", "read from cache failed: "+err.Error())
		}
		return
	}

	if addStatusHeader {
		this.writer.Header().Set("X-Cache", "HIT, "+refType+", "+reader.TypeName())
	}

	// ETag
	var respHeader = this.writer.Header()
	var eTag = respHeader.Get("ETag")
	var lastModifiedAt = reader.LastModified()
	if len(eTag) == 0 {
		if lastModifiedAt > 0 {
			eTag = "\"" + strconv.FormatInt(lastModifiedAt, 10) + "\""
			respHeader["ETag"] = []string{eTag}
		}
	}

	// 支持 Last-Modified
	var modifiedTime = respHeader.Get("Last-Modified")
	if len(modifiedTime) == 0 {
		if lastModifiedAt > 0 {
			modifiedTime = time.Unix(lastModifiedAt, 0).Format("Mon, 02 Jan 2006 15:04:05 GMT")
			if len(respHeader.Get("Last-Modified")) == 0 {
				respHeader.Set("Last-Modified", modifiedTime)
			}
		}
	}

	// 支持 If-None-Match
	if len(eTag) > 0 && this.requestHeader("If-None-Match") == eTag {
		// 自定义Header
		this.processResponseHeaders(http.StatusNotModified)
		this.writer.WriteHeader(http.StatusNotModified)
		this.cacheRef = nil
		return true
	}

	// 支持 If-Modified-Since
	if len(modifiedTime) > 0 && this.requestHeader("If-Modified-Since") == modifiedTime {
		// 自定义Header
		this.processResponseHeaders(http.StatusNotModified)
		this.writer.WriteHeader(http.StatusNotModified)
		this.cacheRef = nil
		return true
	}

	this.processResponseHeaders(reader.Status())

	// 输出Body
	if this.RawReq.Method == http.MethodHead {
		this.writer.WriteHeader(reader.Status())
	} else {
		ifRangeHeaders, ok := this.RawReq.Header["If-Range"]
		supportRange := true
		if ok {
			supportRange = false
			for _, v := range ifRangeHeaders {
				if v == this.writer.Header().Get("ETag") || v == this.writer.Header().Get("Last-Modified") {
					supportRange = true
				}
			}
		}

		// 支持Range
		rangeSet := [][]int64{}
		if supportRange {
			fileSize := reader.BodySize()
			contentRange := this.RawReq.Header.Get("Range")
			if len(contentRange) > 0 {
				if fileSize == 0 {
					this.processResponseHeaders(http.StatusRequestedRangeNotSatisfiable)
					this.writer.WriteHeader(http.StatusRequestedRangeNotSatisfiable)
					return true
				}

				set, ok := httpRequestParseContentRange(contentRange)
				if !ok {
					this.processResponseHeaders(http.StatusRequestedRangeNotSatisfiable)
					this.writer.WriteHeader(http.StatusRequestedRangeNotSatisfiable)
					return true
				}
				if len(set) > 0 {
					rangeSet = set
					for _, arr := range rangeSet {
						if arr[0] == -1 {
							arr[0] = fileSize + arr[1]
							arr[1] = fileSize - 1

							if arr[0] < 0 {
								this.processResponseHeaders(http.StatusRequestedRangeNotSatisfiable)
								this.writer.WriteHeader(http.StatusRequestedRangeNotSatisfiable)
								return true
							}
						}
						if arr[1] < 0 {
							arr[1] = fileSize - 1
						}
						if arr[1] >= fileSize {
							arr[1] = fileSize - 1
						}
						if arr[0] > arr[1] {
							this.processResponseHeaders(http.StatusRequestedRangeNotSatisfiable)
							this.writer.WriteHeader(http.StatusRequestedRangeNotSatisfiable)
							return true
						}
					}
				}
			}
		}

		respHeader := this.writer.Header()
		if len(rangeSet) == 1 {
			respHeader.Set("Content-Range", "bytes "+strconv.FormatInt(rangeSet[0][0], 10)+"-"+strconv.FormatInt(rangeSet[0][1], 10)+"/"+strconv.FormatInt(reader.BodySize(), 10))
			respHeader.Set("Content-Length", strconv.FormatInt(rangeSet[0][1]-rangeSet[0][0]+1, 10))
			this.writer.WriteHeader(http.StatusPartialContent)

			err = reader.ReadBodyRange(buf, rangeSet[0][0], rangeSet[0][1], func(n int) (goNext bool, err error) {
				_, err = this.writer.Write(buf[:n])
				if err != nil {
					return false, err
				}
				return true, nil
			})
			if err != nil {
				if err == caches.ErrInvalidRange {
					this.processResponseHeaders(http.StatusRequestedRangeNotSatisfiable)
					this.writer.WriteHeader(http.StatusRequestedRangeNotSatisfiable)
					return true
				}
				if !this.canIgnore(err) {
					remotelogs.Warn("HTTP_REQUEST_CACHE", "read from cache failed: "+err.Error())
				}
				return
			}
		} else if len(rangeSet) > 1 {
			boundary := httpRequestGenBoundary()
			respHeader.Set("Content-Type", "multipart/byteranges; boundary="+boundary)
			respHeader.Del("Content-Length")
			contentType := respHeader.Get("Content-Type")

			this.writer.WriteHeader(http.StatusPartialContent)

			for index, set := range rangeSet {
				if index == 0 {
					_, err = this.writer.WriteString("--" + boundary + "\r\n")
				} else {
					_, err = this.writer.WriteString("\r\n--" + boundary + "\r\n")
				}
				if err != nil {
					logs.Error(err)
					return true
				}

				_, err = this.writer.WriteString("Content-Range: " + "bytes " + strconv.FormatInt(set[0], 10) + "-" + strconv.FormatInt(set[1], 10) + "/" + strconv.FormatInt(reader.BodySize(), 10) + "\r\n")
				if err != nil {
					logs.Error(err)
					return true
				}

				if len(contentType) > 0 {
					_, err = this.writer.WriteString("Content-Type: " + contentType + "\r\n\r\n")
					if err != nil {
						logs.Error(err)
						return true
					}
				}

				err := reader.ReadBodyRange(buf, set[0], set[1], func(n int) (goNext bool, err error) {
					_, err = this.writer.Write(buf[:n])
					return true, err
				})
				if err != nil {
					if !this.canIgnore(err) {
						remotelogs.Warn("HTTP_REQUEST_CACHE", "read from cache failed: "+err.Error())
					}
					return true
				}
			}

			_, err = this.writer.WriteString("\r\n--" + boundary + "--\r\n")
			if err != nil {
				logs.Error(err)
				return true
			}
		} else { // 没有Range
			this.writer.WriteHeader(reader.Status())

			err = reader.ReadBody(buf, func(n int) (goNext bool, err error) {
				_, err = this.writer.Write(buf[:n])
				if err != nil {
					return false, err
				}
				return true, nil
			})
			if err != nil {
				if !this.canIgnore(err) {
					remotelogs.Warn("HTTP_REQUEST_CACHE", "read from cache failed: "+err.Error())
				}
				return
			}
		}
	}

	this.isCached = true
	this.cacheRef = nil
	return true
}
