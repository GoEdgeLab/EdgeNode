package waf

import (
	"github.com/TeaOSLab/EdgeNode/internal/waf/requests"
	"github.com/iwind/TeaGo/logs"
	"net/http"
)

type GoGroupAction struct {
	GroupId string `yaml:"groupId" json:"groupId"`
}

func (this *GoGroupAction) Perform(waf *WAF, request *requests.Request, writer http.ResponseWriter) (allow bool) {
	group := waf.FindRuleGroup(this.GroupId)
	if group == nil || !group.IsOn {
		return true
	}

	b, set, err := group.MatchRequest(request)
	if err != nil {
		logs.Error(err)
		return true
	}

	if !b {
		return true
	}

	actionObject := FindActionInstance(set.Action, set.ActionOptions)
	if actionObject == nil {
		return true
	}
	return actionObject.Perform(waf, request, writer)
}
