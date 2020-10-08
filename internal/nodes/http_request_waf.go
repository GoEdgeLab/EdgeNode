package nodes

import (
	"github.com/TeaOSLab/EdgeNode/internal/waf"
	"github.com/iwind/TeaGo/logs"
	"net/http"
)

// 调用WAF
func (this *HTTPRequest) doWAFRequest() (blocked bool) {
	w := sharedWAFManager.FindWAF(this.web.FirewallPolicy.Id)
	if w == nil {
		return
	}

	goNext, _, ruleSet, err := w.MatchRequest(this.RawReq, this.writer)
	if err != nil {
		logs.Error(err)
		return
	}

	if ruleSet != nil {
		if ruleSet.Action != waf.ActionAllow {
			// TODO 记录日志
		}
	}

	return !goNext
}

// call response waf
func (this *HTTPRequest) doWAFResponse(resp *http.Response) (blocked bool) {
	w := sharedWAFManager.FindWAF(this.web.FirewallPolicy.Id)
	if w == nil {
		return
	}

	goNext, _, ruleSet, err := w.MatchResponse(this.RawReq, resp, this.writer)
	if err != nil {
		logs.Error(err)
		return
	}

	if ruleSet != nil {
		if ruleSet.Action != waf.ActionAllow {
			// TODO 记录日志
		}
	}

	return !goNext
}
