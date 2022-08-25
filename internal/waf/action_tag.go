package waf

import (
	"github.com/TeaOSLab/EdgeNode/internal/waf/requests"
	"net/http"
)

type TagAction struct {
	BaseAction

	Tags []string `yaml:"tags" json:"tags"`
}

func (this *TagAction) Init(waf *WAF) error {
	return nil
}

func (this *TagAction) Code() string {
	return ActionTag
}

func (this *TagAction) IsAttack() bool {
	return false
}

func (this *TagAction) WillChange() bool {
	return false
}

func (this *TagAction) Perform(waf *WAF, group *RuleGroup, set *RuleSet, request requests.Request, writer http.ResponseWriter) (continueRequest bool, goNextSet bool) {
	return true, true
}
