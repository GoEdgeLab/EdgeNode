// Copyright 2024 GoEdge CDN goedge.cdn@gmail.com. All rights reserved. Official site: https://goedge.cn .
//go:build !plus

package nodes

import "net/http"

func (this *HTTPRequest) processHLSBefore() (blocked bool) {
	//  stub
	return false
}

func (this *HTTPRequest) processM3u8Response(resp *http.Response) error {
	// stub
	return nil
}
