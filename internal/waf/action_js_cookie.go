// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved. Official site: https://goedge.cn .

package waf

import (
	"crypto/md5"
	"fmt"
	"github.com/TeaOSLab/EdgeCommon/pkg/nodeconfigs"
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs/firewallconfigs"
	"github.com/TeaOSLab/EdgeNode/internal/ttlcache"
	"github.com/TeaOSLab/EdgeNode/internal/waf/requests"
	"github.com/iwind/TeaGo/types"
	"net/http"
	"time"
)

type JSCookieAction struct {
	BaseAction

	Life             int32  `yaml:"life" json:"life"`
	MaxFails         int    `yaml:"maxFails" json:"maxFails"`                 // 最大失败次数
	FailBlockTimeout int    `yaml:"failBlockTimeout" json:"failBlockTimeout"` // 失败拦截时间
	Scope            string `yaml:"scope" json:"scope"`
}

func (this *JSCookieAction) Init(waf *WAF) error {
	this.Scope = firewallconfigs.FirewallScopeGlobal

	return nil
}

func (this *JSCookieAction) Code() string {
	return ActionJavascriptCookie
}

func (this *JSCookieAction) IsAttack() bool {
	return false
}

func (this *JSCookieAction) WillChange() bool {
	return true
}

func (this *JSCookieAction) Perform(waf *WAF, group *RuleGroup, set *RuleSet, req requests.Request, writer http.ResponseWriter) (allow bool) {
	// 是否在白名单中
	if SharedIPWhiteList.Contains("set:"+types.String(set.Id), this.Scope, req.WAFServerId(), req.WAFRemoteIP()) {
		return true
	}

	nodeConfig, err := nodeconfigs.SharedNodeConfig()
	if err != nil {
		return true
	}

	var life = this.Life
	if life <= 0 {
		life = 3600
	}

	// 检查Cookie
	var cookieName = "ge_js_validator_" + types.String(set.Id)
	cookie, err := req.WAFRaw().Cookie(cookieName)
	if err == nil && cookie != nil {
		var cookieValue = cookie.Value
		if len(cookieValue) > 10 {
			var timestamp = cookieValue[:10]
			if types.Int64(timestamp) >= time.Now().Unix()-int64(life) && fmt.Sprintf("%x", md5.Sum([]byte(timestamp+"@"+nodeConfig.NodeId))) == cookieValue[10:] {
				return true
			}
		}
	}

	writer.Header().Set("Content-Type", "text/html; charset=utf-8")
	writer.Header().Set("Cache-Control", "no-cache")

	var timestamp = types.String(time.Now().Unix())

	var cookieValue = timestamp + fmt.Sprintf("%x", md5.Sum([]byte(timestamp+"@"+nodeConfig.NodeId)))

	_, _ = writer.Write([]byte(`<!DOCTYPE html>
<html>
<head>
<title></title>
<meta charset="UTF-8"/>
<script type="text/javascript">
document.cookie = "` + cookieName + `=` + cookieValue + `; path=/; max-age=` + types.String(life) + `;";
window.location.reload();
</script>
</head>
<body>
</body>
</html>`))

	// 记录失败次数
	this.increaseFails(req, waf.Id, group.Id, set.Id)

	return false
}

func (this *JSCookieAction) increaseFails(req requests.Request, policyId int64, groupId int64, setId int64) (goNext bool) {
	var maxFails = this.MaxFails
	var failBlockTimeout = this.FailBlockTimeout

	if maxFails <= 0 {
		maxFails = 10 // 默认10次
	} else if maxFails <= 3 {
		maxFails = 3 // 不能小于3，防止意外刷新出现
	}
	if failBlockTimeout <= 0 {
		failBlockTimeout = 1800 // 默认1800s
	}

	var key = "JS_COOKIE:FAILS:" + req.WAFRemoteIP() + ":" + types.String(req.WAFServerId())

	var countFails = ttlcache.SharedCache.IncreaseInt64(key, 1, time.Now().Unix()+300, true)
	if int(countFails) >= maxFails {
		var useLocalFirewall = false

		if this.Scope == firewallconfigs.FirewallScopeGlobal {
			useLocalFirewall = true
		}

		SharedIPBlackList.RecordIP(IPTypeAll, firewallconfigs.FirewallScopeService, req.WAFServerId(), req.WAFRemoteIP(), time.Now().Unix()+int64(failBlockTimeout), policyId, useLocalFirewall, groupId, setId, "JS_COOKIE验证连续失败超过"+types.String(maxFails)+"次")
		return false
	}

	return true
}
