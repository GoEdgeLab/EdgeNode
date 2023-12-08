// Copyright 2023 GoEdge CDN goedge.cdn@gmail.com. All rights reserved. Official site: https://goedge.cn .

package injectionutils

/*
#cgo CFLAGS: -I./libinjection/src

#include <libinjection.h>
#include <stdlib.h>
*/
import "C"
import (
	"net/url"
	"strings"
	"unsafe"
)

// DetectXSS detect XSS in string
func DetectXSS(input string) bool {
	if len(input) == 0 {
		return false
	}

	if detectXSSOne(input) {
		return true
	}

	// 兼容 /PATH?URI
	if (input[0] == '/' || strings.HasPrefix(input, "http://") || strings.HasPrefix(input, "https://")) && len(input) < 4096 {
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
