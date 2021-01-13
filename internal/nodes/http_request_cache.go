package nodes

import (
	"bytes"
	"errors"
	"github.com/TeaOSLab/EdgeNode/internal/caches"
	"github.com/TeaOSLab/EdgeNode/internal/remotelogs"
	"net/http"
	"strconv"
)

// 读取缓存
func (this *HTTPRequest) doCacheRead() (shouldStop bool) {
	if this.web.Cache == nil || !this.web.Cache.IsOn || len(this.web.Cache.CacheRefs) == 0 {
		return
	}

	cachePolicy := sharedNodeConfig.HTTPCachePolicy
	if cachePolicy == nil || !cachePolicy.IsOn {
		return
	}

	// 检查条件
	for _, cacheRef := range this.web.Cache.CacheRefs {
		if !cacheRef.IsOn ||
			cacheRef.Conds == nil ||
			!cacheRef.Conds.HasRequestConds() {
			continue
		}
		if cacheRef.Conds.MatchRequest(this.Format) {
			this.cacheRef = cacheRef
			break
		}
	}
	if this.cacheRef == nil {
		return
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

		remotelogs.Error("REQUEST_CACHE", "read from cache failed: "+err.Error())
		return
	}
	defer func() {
		_ = reader.Close()
	}()

	this.varMapping["cache.status"] = "HIT"

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
		remotelogs.Error("REQUEST_CACHE", "read from cache failed: "+err.Error())
		return
	}

	this.processResponseHeaders(reader.Status())
	this.writer.WriteHeader(reader.Status())

	// 输出Body
	if this.RawReq.Method != http.MethodHead {
		err = reader.ReadBody(buf, func(n int) (goNext bool, err error) {
			_, err = this.writer.Write(buf[:n])
			if err != nil {
				return false, err
			}
			return true, nil
		})
		if err != nil {
			remotelogs.Error("REQUEST_CACHE", "read from cache failed: "+err.Error())
			return
		}
	}

	this.cacheRef = nil // 终止读取不再往下传递
	return true
}
