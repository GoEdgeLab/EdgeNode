package waf

import (
	"github.com/TeaOSLab/EdgeNode/internal/remotelogs"
	"github.com/TeaOSLab/EdgeNode/internal/waf/requests"
	"github.com/iwind/TeaGo/types"
	"net/http"
)

type GoSetAction struct {
	BaseAction

	GroupId string `yaml:"groupId" json:"groupId"`
	SetId   string `yaml:"setId" json:"setId"`
}

func (this *GoSetAction) Init(waf *WAF) error {
	return nil
}

func (this *GoSetAction) Code() string {
	return ActionGoSet
}

func (this *GoSetAction) IsAttack() bool {
	return false
}

func (this *GoSetAction) WillChange() bool {
	return true
}

func (this *GoSetAction) Perform(waf *WAF, group *RuleGroup, set *RuleSet, request requests.Request, writer http.ResponseWriter) (continueRequest bool, goNextSet bool) {
	nextGroup := waf.FindRuleGroup(types.Int64(this.GroupId))
	if nextGroup == nil || !nextGroup.IsOn {
		return true, true
	}
	nextSet := nextGroup.FindRuleSet(types.Int64(this.SetId))
	if nextSet == nil || !nextSet.IsOn {
		return true, true
	}

	b, _, err := nextSet.MatchRequest(request)
	if err != nil {
		remotelogs.Error("WAF", "GO_GROUP_ACTION: "+err.Error())
		return true, false
	}
	if !b {
		return true, false
	}
	return nextSet.PerformActions(waf, nextGroup, request, writer)
}
