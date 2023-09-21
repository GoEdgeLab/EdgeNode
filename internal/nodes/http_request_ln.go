// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved.
//go:build !plus
// +build !plus

package nodes

import (
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs"
)

const (
	LNExpiresHeader = "X-Edge-Ln-Expires"
)

func existsLnNodeIP(nodeIP string) bool {
	return false
}

func (this *HTTPRequest) checkLnRequest() bool {
	return false
}

func (this *HTTPRequest) getLnOrigin(excludingNodeIds []int64, urlHash uint64) (originConfig *serverconfigs.OriginConfig, lnNodeId int64, hasMultipleNodes bool) {
	return nil, 0, false
}
