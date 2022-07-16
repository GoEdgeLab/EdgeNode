package waf

import (
	"github.com/TeaOSLab/EdgeNode/internal/waf/requests"
	"github.com/iwind/TeaGo/logs"
	"github.com/iwind/TeaGo/types"
	"net/http"
)

type GoGroupAction struct {
	BaseAction

	GroupId string `yaml:"groupId" json:"groupId"`
}

func (this *GoGroupAction) Init(waf *WAF) error {
	return nil
}

func (this *GoGroupAction) Code() string {
	return ActionGoGroup
}

func (this *GoGroupAction) IsAttack() bool {
	return false
}

func (this *GoGroupAction) WillChange() bool {
	return true
}

func (this *GoGroupAction) Perform(waf *WAF, group *RuleGroup, set *RuleSet, request requests.Request, writer http.ResponseWriter) (allow bool) {
	nextGroup := waf.FindRuleGroup(types.Int64(this.GroupId))
	if nextGroup == nil || !nextGroup.IsOn {
		return true
	}

	b, _, nextSet, err := nextGroup.MatchRequest(request)
	if err != nil {
		logs.Error(err)
		return true
	}

	if !b {
		return true
	}

	return nextSet.PerformActions(waf, nextGroup, request, writer)
}
