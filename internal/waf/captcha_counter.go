// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved. Official site: https://goedge.cn .

package waf

import (
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs/firewallconfigs"
	"github.com/TeaOSLab/EdgeNode/internal/utils"
	"github.com/TeaOSLab/EdgeNode/internal/utils/counters"
	"github.com/TeaOSLab/EdgeNode/internal/waf/requests"
	"github.com/iwind/TeaGo/types"
	"time"
)

type CaptchaPageCode = string

const (
	CaptchaPageCodeInit   CaptchaPageCode = "init"
	CaptchaPageCodeShow   CaptchaPageCode = "show"
	CaptchaPageCodeSubmit CaptchaPageCode = "submit"
)

// CaptchaIncreaseFails 增加Captcha失败次数，以便后续操作
func CaptchaIncreaseFails(req requests.Request, actionConfig *CaptchaAction, policyId int64, groupId int64, setId int64, pageCode CaptchaPageCode) (goNext bool) {
	var maxFails = actionConfig.MaxFails
	var failBlockTimeout = actionConfig.FailBlockTimeout
	if maxFails > 0 && failBlockTimeout > 0 {
		if maxFails <= 3 {
			maxFails = 3 // 不能小于3，防止意外刷新出现
		}
		var countFails = counters.SharedCounter.IncreaseKey(CaptchaCacheKey(req, pageCode), 300)
		if int(countFails) >= maxFails {
			SharedIPBlackList.RecordIP(IPTypeAll, firewallconfigs.FirewallScopeService, req.WAFServerId(), req.WAFRemoteIP(), time.Now().Unix()+int64(failBlockTimeout), policyId, true, groupId, setId, "CAPTCHA验证连续失败超过"+types.String(maxFails)+"次")
			return false
		}
	}
	return true
}

// CaptchaDeleteCacheKey 清除计数
func CaptchaDeleteCacheKey(req requests.Request) {
	counters.SharedCounter.ResetKey(CaptchaCacheKey(req, CaptchaPageCodeInit))
	counters.SharedCounter.ResetKey(CaptchaCacheKey(req, CaptchaPageCodeShow))
	counters.SharedCounter.ResetKey(CaptchaCacheKey(req, CaptchaPageCodeSubmit))
}

// CaptchaCacheKey 获取Captcha缓存Key
func CaptchaCacheKey(req requests.Request, pageCode CaptchaPageCode) string {
	var requestPath = req.WAFRaw().URL.Path

	if req.WAFRaw().URL.Path == CaptchaPath {
		m, err := utils.SimpleDecryptMap(req.WAFRaw().URL.Query().Get("info"))
		if err == nil && m != nil {
			requestPath = m.GetString("url")
		}
	}

	return "WAF:CAPTCHA:FAILS:" + pageCode + ":" + req.WAFRemoteIP() + ":" + types.String(req.WAFServerId()) + ":" + requestPath
}
