// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved. Official site: https://goedge.cn .

package nodes

import (
	"net/http"
)

const emptyHTTPLocation = "/$EmptyHTTPLocation$"

type HTTPClientTransport struct {
	*http.Transport
}

func (this *HTTPClientTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := this.Transport.RoundTrip(req)
	if err != nil {
		return resp, err
	}

	// 检查在跳转相关状态中Location是否存在
	if httpStatusIsRedirect(resp.StatusCode) && len(resp.Header.Get("Location")) == 0 {
		resp.Header.Set("Location", emptyHTTPLocation)
	}
	return resp, nil
}
