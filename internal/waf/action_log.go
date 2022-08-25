package waf

import (
	"github.com/TeaOSLab/EdgeNode/internal/waf/requests"
	"net/http"
)

type LogAction struct {
	BaseAction
}

func (this *LogAction) Init(waf *WAF) error {
	return nil
}

func (this *LogAction) Code() string {
	return ActionLog
}

func (this *LogAction) IsAttack() bool {
	return false
}

func (this *LogAction) WillChange() bool {
	return false
}

func (this *LogAction) Perform(waf *WAF, group *RuleGroup, set *RuleSet, request requests.Request, writer http.ResponseWriter) (continueRequest bool, goNextSet bool) {
	return true, false
}
