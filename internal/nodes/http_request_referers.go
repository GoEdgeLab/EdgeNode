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

	const cacheSeconds = "3600" // 时间不能过长，防止修改设置后长期无法生效

	var refererURL = this.RawReq.Header.Get("Referer")
	if len(refererURL) == 0 {
		if this.web.Referers.MatchDomain(this.ReqHost, "") {
			return
		}

		this.tags = append(this.tags, "refererCheck")
		this.writer.Header().Set("Cache-Control", "max-age="+cacheSeconds)
		this.writeCode(http.StatusForbidden, "The referer has been blocked.", "当前访问已被防盗链系统拦截。")

		return true
	}

	u, err := url.Parse(refererURL)
	if err != nil {
		if this.web.Referers.MatchDomain(this.ReqHost, "") {
			return
		}

		this.tags = append(this.tags, "refererCheck")
		this.writer.Header().Set("Cache-Control", "max-age="+cacheSeconds)
		this.writeCode(http.StatusForbidden, "The referer has been blocked.", "当前访问已被防盗链系统拦截。")

		return true
	}

	if !this.web.Referers.MatchDomain(this.ReqHost, u.Host) {
		this.tags = append(this.tags, "refererCheck")
		this.writer.Header().Set("Cache-Control", "max-age="+cacheSeconds)
		this.writeCode(http.StatusForbidden, "The referer has been blocked.", "当前访问已被防盗链系统拦截。")
		return true
	}
	return
}
