// Copyright 2023 GoEdge CDN goedge.cdn@gmail.com. All rights reserved. Official site: https://goedge.cn .

package injectionutils

/*
#cgo CFLAGS: -I./libinjection/src

#include <libinjection.h>
#include <stdlib.h>
*/
import "C"
import (
	"github.com/TeaOSLab/EdgeNode/internal/utils/fasttime"
	"github.com/TeaOSLab/EdgeNode/internal/waf/utils"
	"github.com/cespare/xxhash/v2"
	"net/url"
	"strconv"
	"strings"
	"unsafe"
)

func DetectXSSCache(input string, cacheLife utils.CacheLife) bool {
	var l = len(input)

	if l == 0 {
		return false
	}

	if cacheLife <= 0 || l < 512 || l > utils.MaxCacheDataSize {
		return DetectXSS(input)
	}

	var hash = xxhash.Sum64String(input)
	var key = "WAF@XSS@" + strconv.FormatUint(hash, 10)
	var item = utils.SharedCache.Read(key)
	if item != nil {
		return item.Value == 1
	}

	var result = DetectXSS(input)
	if result {
		utils.SharedCache.Write(key, 1, fasttime.Now().Unix()+cacheLife)
	} else {
		utils.SharedCache.Write(key, 0, fasttime.Now().Unix()+cacheLife)
	}
	return result
}

// DetectXSS detect XSS in string
func DetectXSS(input string) bool {
	if len(input) == 0 {
		return false
	}

	if detectXSSOne(input) {
		return true
	}

	// 兼容 /PATH?URI
	if (input[0] == '/' || strings.HasPrefix(input, "http://") || strings.HasPrefix(input, "https://")) && len(input) < 1024 {
		var argsIndex = strings.Index(input, "?")
		if argsIndex > 0 {
			var args = input[argsIndex+1:]
			unescapeArgs, err := url.QueryUnescape(args)
			if err == nil && args != unescapeArgs {
				return detectXSSOne(args) || detectXSSOne(unescapeArgs)
			} else {
				return detectXSSOne(args)
			}
		}
	}

	return false
}

func detectXSSOne(input string) bool {
	if len(input) == 0 {
		return false
	}

	var cInput = C.CString(input)
	defer C.free(unsafe.Pointer(cInput))

	return C.libinjection_xss(cInput, C.size_t(len(input))) == 1
}
