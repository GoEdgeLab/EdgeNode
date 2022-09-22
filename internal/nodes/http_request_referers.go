// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved. Official site: https://goedge.cn .

package nodes

import (
	"net/http"
	"net/url"
)

func (this *HTTPRequest) doCheckReferers() (shouldStop bool) {
	if this.web.Referers == nil {
		return
	}

	var refererURL = this.RawReq.Header.Get("Referer")
	if len(refererURL) == 0 {
		if this.web.Referers.MatchDomain(this.ReqHost, "") {
			return
		}

		this.tags = append(this.tags, "refererCheck")
		this.writer.WriteHeader(http.StatusForbidden)

		return true
	}

	u, err := url.Parse(refererURL)
	if err != nil {
		if this.web.Referers.MatchDomain(this.ReqHost, "") {
			return
		}

		this.tags = append(this.tags, "refererCheck")
		this.writer.WriteHeader(http.StatusForbidden)

		return true
	}

	if !this.web.Referers.MatchDomain(this.ReqHost, u.Host) {
		this.tags = append(this.tags, "refererCheck")
		this.writer.WriteHeader(http.StatusForbidden)
		return true
	}
	return
}
