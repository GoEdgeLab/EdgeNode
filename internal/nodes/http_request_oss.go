// Copyright 2023 GoEdge CDN goedge.cdn@gmail.com. All rights reserved. Official site: https://goedge.cn .
//go:build !plus

package nodes

import (
	"errors"
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs"
	"net/http"
)

func (this *HTTPRequest) doOSSOrigin(origin *serverconfigs.OriginConfig) (resp *http.Response, goNext bool, err error) {
	// stub
	return nil, errors.New("not implemented")
}
