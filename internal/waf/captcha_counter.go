// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved. Official site: https://goedge.cn .

package waf

import (
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs/firewallconfigs"
	"github.com/TeaOSLab/EdgeNode/internal/ttlcache"
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
		var countFails = ttlcache.SharedCache.IncreaseInt64(CaptchaCacheKey(req, pageCode), 1, time.Now().Unix()+300, true)
		if int(countFails) >= maxFails {
			var useLocalFirewall = false

			if actionConfig.FailBlockScopeAll {
				useLocalFirewall = true
			}

			SharedIPBlackList.RecordIP(IPTypeAll, firewallconfigs.FirewallScopeService, req.WAFServerId(), req.WAFRemoteIP(), time.Now().Unix()+int64(failBlockTimeout), policyId, useLocalFirewall, groupId, setId, "CAPTCHA验证连续失败超过"+types.String(maxFails)+"次")
			return false
		}
	}
	return true
}

// CaptchaDeleteCacheKey 清除计数
func CaptchaDeleteCacheKey(req requests.Request) {
	ttlcache.SharedCache.Delete(CaptchaCacheKey(req, CaptchaPageCodeInit))
	ttlcache.SharedCache.Delete(CaptchaCacheKey(req, CaptchaPageCodeShow))
	ttlcache.SharedCache.Delete(CaptchaCacheKey(req, CaptchaPageCodeSubmit))
}

// CaptchaCacheKey 获取Captcha缓存Key
func CaptchaCacheKey(req requests.Request, pageCode CaptchaPageCode) string {
	return "CAPTCHA:FAILS:" + pageCode + ":" + req.WAFRemoteIP() + ":" + types.String(req.WAFServerId())
}
