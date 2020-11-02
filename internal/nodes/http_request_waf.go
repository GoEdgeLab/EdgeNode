package nodes

import (
	"github.com/TeaOSLab/EdgeNode/internal/waf"
	"github.com/iwind/TeaGo/logs"
	"github.com/iwind/TeaGo/types"
	"net/http"
)

// 调用WAF
func (this *HTTPRequest) doWAFRequest() (blocked bool) {
	w := sharedWAFManager.FindWAF(this.web.FirewallPolicy.Id)
	if w == nil {
		return
	}

	goNext, ruleGroup, ruleSet, err := w.MatchRequest(this.RawReq, this.writer)
	if err != nil {
		logs.Error(err)
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
		logs.Error(err)
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
