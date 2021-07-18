package waf

import (
	"github.com/TeaOSLab/EdgeNode/internal/remotelogs"
	"github.com/TeaOSLab/EdgeNode/internal/waf/requests"
	"github.com/iwind/TeaGo/lists"
	"github.com/iwind/TeaGo/logs"
	"github.com/iwind/TeaGo/maps"
	"github.com/iwind/TeaGo/utils/string"
	"net/http"
)

type RuleConnector = string

const (
	RuleConnectorAnd = "and"
	RuleConnectorOr  = "or"
)

type RuleSet struct {
	Id          string          `yaml:"id" json:"id"`
	Code        string          `yaml:"code" json:"code"`
	IsOn        bool            `yaml:"isOn" json:"isOn"`
	Name        string          `yaml:"name" json:"name"`
	Description string          `yaml:"description" json:"description"`
	Rules       []*Rule         `yaml:"rules" json:"rules"`
	Connector   RuleConnector   `yaml:"connector" json:"connector"` // rules connector
	Actions     []*ActionConfig `yaml:"actions" json:"actions"`

	actionCodes     []string
	actionInstances []ActionInterface

	hasRules bool
}

func NewRuleSet() *RuleSet {
	return &RuleSet{
		Id:   stringutil.Rand(16),
		IsOn: true,
	}
}

func (this *RuleSet) Init(waf *WAF) error {
	this.hasRules = len(this.Rules) > 0
	if this.hasRules {
		for _, rule := range this.Rules {
			err := rule.Init()
			if err != nil {
				return err
			}
		}
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
		instance := FindActionInstance(action.Code, action.Options)
		if instance == nil {
			remotelogs.Error("WAF_RULE_SET", "can not find instance for action '"+action.Code+"'")
		} else {
			this.actionInstances = append(this.actionInstances, instance)
		}
		err := instance.Init(waf)
		if err != nil {
			remotelogs.Error("WAF_RULE_SET", "init action '"+action.Code+"' failed: "+err.Error())
		}
	}

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

func (this *RuleSet) PerformActions(waf *WAF, group *RuleGroup, req requests.Request, writer http.ResponseWriter) bool {
	// 先执行allow
	for _, instance := range this.actionInstances {
		if !instance.WillChange() {
			if waf.onActionCallback != nil {
				goNext := waf.onActionCallback(instance)
				if !goNext {
					return false
				}
			}
			logs.Printf("perform1: %#v", instance) // TODO
			instance.Perform(waf, group, this, req, writer)
		}
	}

	// 再执行block|verify
	for _, instance := range this.actionInstances {
		// 只执行第一个可能改变请求的动作，其余的都会被忽略
		if instance.WillChange() {
			if waf.onActionCallback != nil {
				goNext := waf.onActionCallback(instance)
				if !goNext {
					return false
				}
			}
			logs.Printf("perform2: %#v", instance) // TODO
			return instance.Perform(waf, group, this, req, writer)
		}
	}

	return true
}

func (this *RuleSet) MatchRequest(req requests.Request) (b bool, err error) {
	if !this.hasRules {
		return false, nil
	}
	switch this.Connector {
	case RuleConnectorAnd:
		for _, rule := range this.Rules {
			b1, err1 := rule.MatchRequest(req)
			if err1 != nil {
				return false, err1
			}
			if !b1 {
				return false, nil
			}
		}
		return true, nil
	case RuleConnectorOr:
		for _, rule := range this.Rules {
			b1, err1 := rule.MatchRequest(req)
			if err1 != nil {
				return false, err1
			}
			if b1 {
				return true, nil
			}
		}
	default: // same as And
		for _, rule := range this.Rules {
			b1, err1 := rule.MatchRequest(req)
			if err1 != nil {
				return false, err1
			}
			if !b1 {
				return false, nil
			}
		}
		return true, nil
	}
	return
}

func (this *RuleSet) MatchResponse(req requests.Request, resp *requests.Response) (b bool, err error) {
	if !this.hasRules {
		return false, nil
	}
	switch this.Connector {
	case RuleConnectorAnd:
		for _, rule := range this.Rules {
			b1, err1 := rule.MatchResponse(req, resp)
			if err1 != nil {
				return false, err1
			}
			if !b1 {
				return false, nil
			}
		}
		return true, nil
	case RuleConnectorOr:
		for _, rule := range this.Rules {
			b1, err1 := rule.MatchResponse(req, resp)
			if err1 != nil {
				return false, err1
			}
			if b1 {
				return true, nil
			}
		}
	default: // same as And
		for _, rule := range this.Rules {
			b1, err1 := rule.MatchResponse(req, resp)
			if err1 != nil {
				return false, err1
			}
			if !b1 {
				return false, nil
			}
		}
		return true, nil
	}
	return
}
