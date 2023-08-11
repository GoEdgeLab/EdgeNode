package waf

import (
	"fmt"
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs/firewallconfigs"
	"github.com/TeaOSLab/EdgeNode/internal/remotelogs"
	"github.com/TeaOSLab/EdgeNode/internal/utils"
	"github.com/TeaOSLab/EdgeNode/internal/waf/requests"
	"github.com/iwind/TeaGo/lists"
	"github.com/iwind/TeaGo/maps"
	"github.com/iwind/TeaGo/types"
	"net/http"
	"sort"
)

type RuleConnector = string

const (
	RuleConnectorAnd = "and"
	RuleConnectorOr  = "or"
)

type RuleSet struct {
	Id          int64           `yaml:"id" json:"id"`
	Code        string          `yaml:"code" json:"code"`
	IsOn        bool            `yaml:"isOn" json:"isOn"`
	Name        string          `yaml:"name" json:"name"`
	Description string          `yaml:"description" json:"description"`
	Rules       []*Rule         `yaml:"rules" json:"rules"`
	Connector   RuleConnector   `yaml:"connector" json:"connector"` // rules connector
	Actions     []*ActionConfig `yaml:"actions" json:"actions"`
	IgnoreLocal bool            `yaml:"ignoreLocal" json:"ignoreLocal"`

	actionCodes     []string
	actionInstances []ActionInterface

	hasRules bool
}

func NewRuleSet() *RuleSet {
	return &RuleSet{
		IsOn: true,
	}
}

func (this *RuleSet) Init(waf *WAF) error {
	this.hasRules = len(this.Rules) > 0
	if this.hasRules {
		for _, rule := range this.Rules {
			err := rule.Init()
			if err != nil {
				return fmt.Errorf("init rule '%s %s %s' failed: %w", rule.Param, rule.Operator, types.String(rule.Value), err)
			}
		}

		// sort by priority
		sort.Slice(this.Rules, func(i, j int) bool {
			return this.Rules[i].Priority > this.Rules[j].Priority
		})
	}

	// action codes
	var actionCodes = []string{}
	for _, action := range this.Actions {
		if !lists.ContainsString(actionCodes, action.Code) {
			actionCodes = append(actionCodes, action.Code)
		}
	}
	this.actionCodes = actionCodes

	// action instances
	this.actionInstances = []ActionInterface{}
	for _, action := range this.Actions {
		var instance = FindActionInstance(action.Code, action.Options)
		if instance == nil {
			remotelogs.Error("WAF_RULE_SET", "can not find instance for action '"+action.Code+"'")
			continue
		}

		err := instance.Init(waf)
		if err != nil {
			remotelogs.Error("WAF_RULE_SET", "init action '"+action.Code+"' failed: "+err.Error())
			continue
		}

		this.actionInstances = append(this.actionInstances, instance)
		waf.AddAction(instance)
	}

	// sort actions
	sort.Slice(this.actionInstances, func(i, j int) bool {
		var instance1 = this.actionInstances[i]
		if !instance1.WillChange() {
			return true
		}
		if instance1.Code() == ActionRecordIP {
			return true
		}
		return false
	})

	return nil
}

func (this *RuleSet) AddRule(rule ...*Rule) {
	this.Rules = append(this.Rules, rule...)
}

// AddAction 添加动作
func (this *RuleSet) AddAction(code string, options maps.Map) {
	if options == nil {
		options = maps.Map{}
	}
	this.Actions = append(this.Actions, &ActionConfig{
		Code:    code,
		Options: options,
	})
}

// HasSpecialActions 除了Allow之外是否还有别的动作
func (this *RuleSet) HasSpecialActions() bool {
	for _, action := range this.Actions {
		if action.Code != ActionAllow {
			return true
		}
	}
	return false
}

// HasAttackActions 检查是否含有攻击防御动作
func (this *RuleSet) HasAttackActions() bool {
	for _, action := range this.actionInstances {
		if action.IsAttack() {
			return true
		}
	}
	return false
}

func (this *RuleSet) ActionCodes() []string {
	return this.actionCodes
}

func (this *RuleSet) PerformActions(waf *WAF, group *RuleGroup, req requests.Request, writer http.ResponseWriter) (continueRequest bool, goNextSet bool) {
	if len(waf.Mode) != 0 && waf.Mode != firewallconfigs.FirewallModeDefend {
		return true, false
	}

	// 先执行allow
	for _, instance := range this.actionInstances {
		if !instance.WillChange() {
			continueRequest = req.WAFOnAction(instance)
			if !continueRequest {
				return false, false
			}
			_, goNextSet = instance.Perform(waf, group, this, req, writer)
		}
	}

	// 再执行block|verify
	for _, instance := range this.actionInstances {
		// 只执行第一个可能改变请求的动作，其余的都会被忽略
		if instance.WillChange() {
			continueRequest = req.WAFOnAction(instance)
			if !continueRequest {
				return false, false
			}
			return instance.Perform(waf, group, this, req, writer)
		}
	}

	return true, goNextSet
}

func (this *RuleSet) MatchRequest(req requests.Request) (b bool, hasRequestBody bool, err error) {
	// 是否忽略局域网IP
	if this.IgnoreLocal && utils.IsLocalIP(req.WAFRemoteIP()) {
		return false, hasRequestBody, nil
	}

	if !this.hasRules {
		return false, hasRequestBody, nil
	}
	switch this.Connector {
	case RuleConnectorAnd:
		for _, rule := range this.Rules {
			b1, hasCheckRequestBody, err1 := rule.MatchRequest(req)
			if hasCheckRequestBody {
				hasRequestBody = true
			}
			if err1 != nil {
				return false, hasRequestBody, err1
			}
			if !b1 {
				return false, hasRequestBody, nil
			}
		}
		return true, hasRequestBody, nil
	case RuleConnectorOr:
		for _, rule := range this.Rules {
			b1, hasCheckRequestBody, err1 := rule.MatchRequest(req)
			if hasCheckRequestBody {
				hasRequestBody = true
			}
			if err1 != nil {
				return false, hasRequestBody, err1
			}
			if b1 {
				return true, hasRequestBody, nil
			}
		}
	default: // same as And
		for _, rule := range this.Rules {
			b1, hasCheckRequestBody, err1 := rule.MatchRequest(req)
			if hasCheckRequestBody {
				hasRequestBody = true
			}
			if err1 != nil {
				return false, hasRequestBody, err1
			}
			if !b1 {
				return false, hasRequestBody, nil
			}
		}
		return true, hasRequestBody, nil
	}
	return
}

func (this *RuleSet) MatchResponse(req requests.Request, resp *requests.Response) (b bool, hasRequestBody bool, err error) {
	if !this.hasRules {
		return false, hasRequestBody, nil
	}
	switch this.Connector {
	case RuleConnectorAnd:
		for _, rule := range this.Rules {
			b1, hasCheckRequestBody, err1 := rule.MatchResponse(req, resp)
			if hasCheckRequestBody {
				hasRequestBody = true
			}
			if err1 != nil {
				return false, hasRequestBody, err1
			}
			if !b1 {
				return false, hasRequestBody, nil
			}
		}
		return true, hasRequestBody, nil
	case RuleConnectorOr:
		for _, rule := range this.Rules {
			// 对于OR连接符，只需要判断最先匹配的一条规则中的hasRequestBody即可
			b1, hasCheckRequestBody, err1 := rule.MatchResponse(req, resp)
			if err1 != nil {
				return false, hasCheckRequestBody, err1
			}
			if b1 {
				return true, hasCheckRequestBody, nil
			}
		}
	default: // same as And
		for _, rule := range this.Rules {
			b1, hasCheckRequestBody, err1 := rule.MatchResponse(req, resp)
			if hasCheckRequestBody {
				hasRequestBody = true
			}
			if err1 != nil {
				return false, hasRequestBody, err1
			}
			if !b1 {
				return false, hasRequestBody, nil
			}
		}
		return true, hasRequestBody, nil
	}
	return
}
