package waf

import (
	"github.com/TeaOSLab/EdgeNode/internal/waf/requests"
	"net/http"
)

type AllowAction struct {
	BaseAction
}

func (this *AllowAction) Init(waf *WAF) error {
	return nil
}

func (this *AllowAction) Code() string {
	return ActionAllow
}

func (this *AllowAction) IsAttack() bool {
	return false
}

func (this *AllowAction) WillChange() bool {
	return true
}

func (this *AllowAction) Perform(waf *WAF, group *RuleGroup, set *RuleSet, request requests.Request, writer http.ResponseWriter) (continueRequest bool, goNextSet bool) {
	// do nothing
	return true, false
}
