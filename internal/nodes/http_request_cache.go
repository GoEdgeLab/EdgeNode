package nodes

import (
	"bytes"
	"github.com/TeaOSLab/EdgeNode/internal/caches"
	"github.com/iwind/TeaGo/logs"
	"github.com/iwind/TeaGo/types"
	"net/http"
	"strconv"
)

// 读取缓存
func (this *HTTPRequest) doCacheRead() (shouldStop bool) {
	if this.web.Cache == nil || !this.web.Cache.IsOn || len(this.web.Cache.CacheRefs) == 0 {
		return
	}

	// 检查条件
	for _, cacheRef := range this.web.Cache.CacheRefs {
		if !cacheRef.IsOn ||
			cacheRef.CachePolicyId == 0 ||
			cacheRef.CachePolicy == nil ||
			!cacheRef.CachePolicy.IsOn ||
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
	this.varMapping["cache.policy.name"] = this.cacheRef.CachePolicy.Name
	this.varMapping["cache.policy.id"] = strconv.FormatInt(this.cacheRef.CachePolicy.Id, 10)
	this.varMapping["cache.policy.type"] = this.cacheRef.CachePolicy.Type

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
	storage := caches.SharedManager.FindStorageWithPolicy(this.cacheRef.CachePolicyId)
	if storage == nil {
		this.cacheRef = nil
		return
	}

	isBroken := false
	headerBuf := []byte{}
	statusCode := http.StatusOK
	statusFound := false
	headerFound := false

	buf := bytePool32k.Get()
	err := storage.Read(key, buf, func(data []byte, valueSize int64, expiredAt int64, isEOF bool) {
		if isBroken {
			return
		}

		// 如果Header已发送完毕
		if headerFound {
			_, _ = this.writer.Write(data)
			return
		}

		headerBuf = append(headerBuf, data...)

		if !statusFound {
			lineIndex := bytes.IndexByte(headerBuf, '\n')
			if lineIndex < 0 {
				return
			}

			pieces := bytes.Split(headerBuf[:lineIndex], []byte{' '})
			if len(pieces) < 2 {
				isBroken = true
				return
			}
			statusCode = types.Int(string(pieces[1]))
			statusFound = true
			headerBuf = headerBuf[lineIndex+1:]

			// cache相关变量
			this.varMapping["cache.status"] = "HIT"
		}

		for {
			lineIndex := bytes.IndexByte(headerBuf, '\n')
			if lineIndex < 0 {
				break
			}
			if lineIndex == 0 || lineIndex == 1 {
				headerFound = true

				this.processResponseHeaders(statusCode)
				this.writer.WriteHeader(statusCode)

				_, _ = this.writer.Write(headerBuf[lineIndex+1:])
				headerBuf = nil
				break
			}

			// 分解Header
			line := headerBuf[:lineIndex]
			colonIndex := bytes.IndexByte(line, ':')
			if colonIndex <= 0 {
				continue
			}
			this.writer.Header().Set(string(line[:colonIndex]), string(bytes.TrimSpace(line[colonIndex+1:])))
			headerBuf = headerBuf[lineIndex+1:]
		}
	})

	bytePool32k.Put(buf)

	if err != nil {
		if err == caches.ErrNotFound {
			// cache相关变量
			this.varMapping["cache.status"] = "MISS"
			return
		}

		logs.Println("read from cache failed: " + err.Error())
		return
	}

	if isBroken {
		return
	}

	return true
}
