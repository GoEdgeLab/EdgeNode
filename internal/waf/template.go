package waf

import (
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs/firewallconfigs"
	"github.com/TeaOSLab/EdgeNode/internal/waf/utils"
)

func Template() (*WAF, error) {
	var config = firewallconfigs.HTTPFirewallTemplate()
	if config.Inbound != nil {
		config.Inbound.IsOn = true
	}

	for _, group := range config.AllRuleGroups() {
		if group.Code == "cc" || group.Code == "cc2" {
			continue
		}
		group.IsOn = true

		for _, set := range group.Sets {
			set.IsOn = true
		}
	}

	instance, err := SharedWAFManager.ConvertWAF(config)
	if err != nil {
		return nil, err
	}

	for _, group := range instance.Inbound {
		for _, set := range group.RuleSets {
			for _, rule := range set.Rules {
				rule.cacheLife = utils.CacheDisabled // for performance test
				_ = rule
			}
		}
	}

	return instance, nil
}
