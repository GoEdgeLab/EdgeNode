// Copyright 2023 GoEdge CDN goedge.cdn@gmail.com. All rights reserved. Official site: https://goedge.cn .
//go:build !plus

package nodes

import (
	"errors"
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs"
	"net/http"
)

func (this *HTTPRequest) doOSSOrigin(origin *serverconfigs.OriginConfig) (resp *http.Response, goNext bool, errorCode string, ossBucketName string, err error) {
	// stub
	return nil, false, "", "", errors.New("not implemented")
}
