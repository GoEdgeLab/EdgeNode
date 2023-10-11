// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package checkpoints

import (
	"github.com/TeaOSLab/EdgeCommon/pkg/configutils"
	"github.com/TeaOSLab/EdgeNode/internal/waf/requests"
	"github.com/TeaOSLab/EdgeNode/internal/waf/utils"
	"github.com/iwind/TeaGo/maps"
	"github.com/iwind/TeaGo/types"
	"net/url"
)

// RequestRefererBlockCheckpoint 防盗链
type RequestRefererBlockCheckpoint struct {
	Checkpoint
}

// RequestValue 计算checkpoint值
// 选项：allowEmpty, allowSameDomain, allowDomains
func (this *RequestRefererBlockCheckpoint) RequestValue(req requests.Request, param string, options maps.Map, ruleId int64) (value any, hasRequestBody bool, sysErr error, userErr error) {
	var checkOrigin = options.GetBool("checkOrigin")
	var referer = req.WAFRaw().Referer()
	if len(referer) == 0 && checkOrigin {
		var origin = req.WAFRaw().Header.Get("Origin")
		if len(origin) > 0 && origin != "null" {
			referer = "https://" + origin // 因为Origin都只有域名部分，所以为了下面的URL 分析需要加上https://
		}
	}

	if len(referer) == 0 {
		if options.GetBool("allowEmpty") {
			value = 1
			return
		}
		value = 0
		return
	}

	u, err := url.Parse(referer)
	if err != nil {
		value = 0
		return
	}
	var host = u.Host

	if options.GetBool("allowSameDomain") && host == req.WAFRaw().Host {
		value = 1
		return
	}

	// allow domains
	var allowDomains = options.GetSlice("allowDomains")
	var allowDomainStrings = []string{}
	for _, domain := range allowDomains {
		allowDomainStrings = append(allowDomainStrings, types.String(domain))
	}

	// deny domains
	var denyDomains = options.GetSlice("denyDomains")
	var denyDomainStrings = []string{}
	for _, domain := range denyDomains {
		denyDomainStrings = append(denyDomainStrings, types.String(domain))
	}

	if len(allowDomainStrings) == 0 {
		if len(denyDomainStrings) > 0 {
			if configutils.MatchDomains(denyDomainStrings, host) {
				value = 0
			} else {
				value = 1
			}
			return
		}

		value = 0
		return
	}

	if configutils.MatchDomains(allowDomainStrings, host) {
		if len(denyDomainStrings) > 0 {
			if configutils.MatchDomains(denyDomainStrings, host) {
				value = 0
			} else {
				value = 1
			}
			return
		}
		value = 1
		return
	} else {
		value = 0
	}

	return
}

func (this *RequestRefererBlockCheckpoint) ResponseValue(req requests.Request, resp *requests.Response, param string, options maps.Map, ruleId int64) (value any, hasRequestBody bool, sysErr error, userErr error) {
	return
}

func (this *RequestRefererBlockCheckpoint) CacheLife() utils.CacheLife {
	return utils.CacheLongLife
}
