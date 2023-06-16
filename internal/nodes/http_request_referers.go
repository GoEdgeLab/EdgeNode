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

	var origin = this.RawReq.Header.Get("Origin")

	const cacheSeconds = "3600" // 时间不能过长，防止修改设置后长期无法生效

	// 处理用到Origin的特殊功能
	if this.web.Referers.CheckOrigin && len(origin) > 0 {
		// 处理Websocket
		if this.web.Websocket != nil && this.web.Websocket.IsOn && this.RawReq.Header.Get("Upgrade") == "websocket" {
			originHost, _ := httpParseHost(origin)
			if len(originHost) > 0 && this.web.Websocket.MatchOrigin(originHost) {
				return
			}
		}
	}

	var refererURL = this.RawReq.Header.Get("Referer")
	if len(refererURL) == 0 && this.web.Referers.CheckOrigin {
		if len(origin) > 0 && origin != "null" {
			if urlSchemeRegexp.MatchString(origin) {
				refererURL = origin
			} else {
				refererURL = "https://" + origin
			}
		}
	}

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
