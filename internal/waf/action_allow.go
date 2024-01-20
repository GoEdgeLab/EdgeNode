package waf

import (
	"github.com/TeaOSLab/EdgeNode/internal/waf/requests"
	"net/http"
)

type AllowScope = string

const (
	AllowScopeGroup  AllowScope = "group"
	AllowScopeServer AllowScope = "server"
	AllowScopeGlobal AllowScope = "global"
)

type AllowAction struct {
	BaseAction

	Scope AllowScope `yaml:"scope" json:"scope"`
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

func (this *AllowAction) Perform(waf *WAF, group *RuleGroup, set *RuleSet, request requests.Request, writer http.ResponseWriter) PerformResult {
	// do nothing
	return PerformResult{
		ContinueRequest: true,
		GoNextGroup:     this.Scope == AllowScopeGroup,
		IsAllowed:       true,
		AllowScope:      this.Scope,
	}
}
