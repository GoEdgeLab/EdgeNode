package waf

import (
	"github.com/TeaOSLab/EdgeNode/internal/waf/requests"
	"net/http"
)

type AllowAction struct {
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
	return false
}

func (this *AllowAction) Perform(waf *WAF, group *RuleGroup, set *RuleSet, request requests.Request, writer http.ResponseWriter) (allow bool) {
	// do nothing
	return true
}
