package nodes

import (
	"github.com/TeaOSLab/EdgeNode/internal/iplibrary"
	"github.com/TeaOSLab/EdgeNode/internal/logs"
	"github.com/TeaOSLab/EdgeNode/internal/waf"
	"github.com/iwind/TeaGo/lists"
	"github.com/iwind/TeaGo/types"
	"net/http"
)

// 调用WAF
func (this *HTTPRequest) doWAFRequest() (blocked bool) {
	// 检查配置是否为空
	if this.web.FirewallPolicy == nil || this.web.FirewallPolicy.Inbound == nil || !this.web.FirewallPolicy.Inbound.IsOn {
		return
	}

	// 检查IP白名单
	remoteAddr := this.requestRemoteAddr()
	inbound := this.web.FirewallPolicy.Inbound
	if inbound.WhiteListRef != nil && inbound.WhiteListRef.IsOn && inbound.WhiteListRef.ListId > 0 {
		list := iplibrary.SharedIPListManager.FindList(inbound.WhiteListRef.ListId)
		if list != nil && list.Contains(iplibrary.IP2Long(remoteAddr)) {
			return
		}
	}

	// 检查IP黑名单
	if inbound.BlackListRef != nil && inbound.BlackListRef.IsOn && inbound.BlackListRef.ListId > 0 {
		list := iplibrary.SharedIPListManager.FindList(inbound.BlackListRef.ListId)
		if list != nil && list.Contains(iplibrary.IP2Long(remoteAddr)) {
			// TODO 可以配置对封禁的处理方式等
			this.writer.WriteHeader(http.StatusForbidden)
			this.writer.Close()

			// 停止日志
			this.disableLog = true

			return true
		}
	}

	// 检查地区封禁
	if iplibrary.SharedLibrary != nil {
		if this.web.FirewallPolicy.Inbound.Region != nil && this.web.FirewallPolicy.Inbound.Region.IsOn {
			regionConfig := this.web.FirewallPolicy.Inbound.Region
			if regionConfig.IsNotEmpty() {
				result, err := iplibrary.SharedLibrary.Lookup(remoteAddr)
				if err != nil {
					logs.Error("REQUEST", "iplibrary lookup failed: "+err.Error())
				} else if result != nil {
					// 检查国家级别封禁
					if len(regionConfig.DenyCountryIds) > 0 && len(result.Country) > 0 {
						countryId := iplibrary.SharedCountryManager.Lookup(result.Country)
						if countryId > 0 && lists.ContainsInt64(regionConfig.DenyCountryIds, countryId) {
							// TODO 可以配置对封禁的处理方式等
							this.writer.WriteHeader(http.StatusForbidden)
							this.writer.Close()

							// 停止日志
							this.disableLog = true

							return true
						}
					}

					// 检查省份封禁
					if len(regionConfig.DenyProvinceIds) > 0 && len(result.Province) > 0 {
						provinceId := iplibrary.SharedProvinceManager.Lookup(result.Province)
						if provinceId > 0 && lists.ContainsInt64(regionConfig.DenyProvinceIds, provinceId) {
							// TODO 可以配置对封禁的处理方式等
							this.writer.WriteHeader(http.StatusForbidden)
							this.writer.Close()

							// 停止日志
							this.disableLog = true

							return true
						}
					}
				}
			}
		}
	}

	// 规则测试
	w := sharedWAFManager.FindWAF(this.web.FirewallPolicy.Id)
	if w == nil {
		return
	}
	goNext, ruleGroup, ruleSet, err := w.MatchRequest(this.RawReq, this.writer)
	if err != nil {
		logs.Error("REQUEST", this.rawURI+": "+err.Error())
		return
	}

	if ruleSet != nil {
		if ruleSet.Action != waf.ActionAllow {
			this.firewallPolicyId = this.web.FirewallPolicy.Id
			this.firewallRuleGroupId = types.Int64(ruleGroup.Id)
			this.firewallRuleSetId = types.Int64(ruleSet.Id)
		}

		this.logAttrs["waf.action"] = ruleSet.Action
	}

	return !goNext
}

// call response waf
func (this *HTTPRequest) doWAFResponse(resp *http.Response) (blocked bool) {
	w := sharedWAFManager.FindWAF(this.web.FirewallPolicy.Id)
	if w == nil {
		return
	}

	goNext, ruleGroup, ruleSet, err := w.MatchResponse(this.RawReq, resp, this.writer)
	if err != nil {
		logs.Error("REQUEST", this.rawURI+": "+err.Error())
		return
	}

	if ruleSet != nil {
		if ruleSet.Action != waf.ActionAllow {
			this.firewallPolicyId = this.web.FirewallPolicy.Id
			this.firewallRuleGroupId = types.Int64(ruleGroup.Id)
			this.firewallRuleSetId = types.Int64(ruleSet.Id)
		}

		this.logAttrs["waf.action"] = ruleSet.Action
	}

	return !goNext
}
