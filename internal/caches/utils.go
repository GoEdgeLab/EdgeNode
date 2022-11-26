// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved. Official site: https://goedge.cn .

package caches

import (
	"github.com/TeaOSLab/EdgeCommon/pkg/configutils"
	"net"
	"strings"
)

func ParseHost(key string) string {
	var schemeIndex = strings.Index(key, "://")
	if schemeIndex <= 0 {
		return ""
	}

	var firstSlashIndex = strings.Index(key[schemeIndex+3:], "/")
	if firstSlashIndex <= 0 {
		return ""
	}

	var host = key[schemeIndex+3 : schemeIndex+3+firstSlashIndex]

	hostPart, _, err := net.SplitHostPort(host)
	if err == nil && len(hostPart) > 0 {
		host = configutils.QuoteIP(hostPart)
	}

	return host
}
