// Copyright 2023 GoEdge CDN goedge.cdn@gmail.com. All rights reserved. Official site: https://goedge.cn .

package injectionutils

/*
#cgo CFLAGS: -I./libinjection/src

#include <libinjection.h>
#include <libinjection_sqli.h>
#include <stdlib.h>
*/
import "C"
import (
	"net/url"
	"strings"
	"unsafe"
)

// DetectSQLInjection detect sql injection in string
func DetectSQLInjection(input string) bool {
	if len(input) == 0 {
		return false
	}

	if detectSQLInjectionOne(input) {
		return true
	}

	// 兼容 /PATH?URI
	if input[0] == '/' {
		var argsIndex = strings.Index(input, "?")
		if argsIndex > 0 {
			var args = input[argsIndex+1:]
			unescapeArgs, err := url.QueryUnescape(args)
			if err == nil && args != unescapeArgs {
				return detectSQLInjectionOne(args) || detectSQLInjectionOne(unescapeArgs)
			} else {
				return detectSQLInjectionOne(args)
			}
		}
	}

	return false
}

func detectSQLInjectionOne(input string) bool {
	if len(input) == 0 {
		return false
	}

	var fingerprint [8]C.char
	var fingerprintPtr = (*C.char)(unsafe.Pointer(&fingerprint[0]))
	var cInput = C.CString(input)
	defer C.free(unsafe.Pointer(cInput))

	return C.libinjection_sqli(cInput, C.size_t(len(input)), fingerprintPtr) == 1
}
