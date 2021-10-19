// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package checkpoints

import (
	"github.com/TeaOSLab/EdgeCommon/pkg/configutils"
	"github.com/TeaOSLab/EdgeNode/internal/waf/requests"
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
func (this *RequestRefererBlockCheckpoint) RequestValue(req requests.Request, param string, options maps.Map) (value interface{}, sysErr error, userErr error) {
	var referer = req.WAFRaw().Referer()

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

	var domains = options.GetSlice("allowDomains")
	var domainStrings = []string{}
	for _, domain := range domains {
		domainStrings = append(domainStrings, types.String(domain))
	}

	if len(domainStrings) == 0 {
		value = 0
		return
	}

	if configutils.MatchDomains(domainStrings, host) {
		value = 1
	} else {
		value = 0
	}

	return
}

func (this *RequestRefererBlockCheckpoint) ResponseValue(req requests.Request, resp *requests.Response, param string, options maps.Map) (value interface{}, sysErr error, userErr error) {
	return
}
