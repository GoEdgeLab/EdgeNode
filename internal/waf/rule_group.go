package waf

import (
	"fmt"
	"github.com/TeaOSLab/EdgeNode/internal/waf/requests"
)

// rule group
type RuleGroup struct {
	Id          int64      `yaml:"id" json:"id"`
	IsOn        bool       `yaml:"isOn" json:"isOn"`
	Name        string     `yaml:"name" json:"name"` // such as SQL Injection
	Description string     `yaml:"description" json:"description"`
	Code        string     `yaml:"code" json:"code"` // identify the group
	RuleSets    []*RuleSet `yaml:"ruleSets" json:"ruleSets"`
	IsInbound   bool       `yaml:"isInbound" json:"isInbound"`

	hasRuleSets bool
}

func NewRuleGroup() *RuleGroup {
	return &RuleGroup{
		IsOn: true,
	}
}

func (this *RuleGroup) Init(waf *WAF) error {
	this.hasRuleSets = len(this.RuleSets) > 0

	if this.hasRuleSets {
		for _, set := range this.RuleSets {
			err := set.Init(waf)
			if err != nil {
				return fmt.Errorf("init set '%d' failed: %w", set.Id, err)
			}
		}
	}
	return nil
}

func (this *RuleGroup) AddRuleSet(ruleSet *RuleSet) {
	this.RuleSets = append(this.RuleSets, ruleSet)
}

func (this *RuleGroup) FindRuleSet(id int64) *RuleSet {
	for _, ruleSet := range this.RuleSets {
		if ruleSet.Id == id {
			return ruleSet
		}
	}
	return nil
}

func (this *RuleGroup) FindRuleSetWithCode(code string) *RuleSet {
	if len(code) == 0 {
		return nil
	}
	for _, ruleSet := range this.RuleSets {
		if ruleSet.Code == code {
			return ruleSet
		}
	}
	return nil
}

func (this *RuleGroup) RemoveRuleSet(id int64) {
	result := []*RuleSet{}
	for _, ruleSet := range this.RuleSets {
		if ruleSet.Id == id {
			continue
		}
		result = append(result, ruleSet)
	}
	this.RuleSets = result
}

func (this *RuleGroup) MatchRequest(req requests.Request) (b bool, hasRequestBody bool, set *RuleSet, err error) {
	if !this.hasRuleSets {
		return
	}
	for _, set := range this.RuleSets {
		if !set.IsOn {
			continue
		}
		b, hasRequestBody, err = set.MatchRequest(req)
		if err != nil {
			return false, hasRequestBody, nil, err
		}
		if b {
			return true, hasRequestBody, set, nil
		}
	}
	return
}

func (this *RuleGroup) MatchResponse(req requests.Request, resp *requests.Response) (b bool, hasRequestBody bool, set *RuleSet, err error) {
	if !this.hasRuleSets {
		return
	}
	for _, set := range this.RuleSets {
		if !set.IsOn {
			continue
		}
		b, hasRequestBody, err = set.MatchResponse(req, resp)
		if err != nil {
			return false, hasRequestBody, nil, err
		}
		if b {
			return true, hasRequestBody, set, nil
		}
	}
	return
}

func (this *RuleGroup) MoveRuleSet(fromIndex int, toIndex int) {
	if fromIndex < 0 || fromIndex >= len(this.RuleSets) {
		return
	}
	if toIndex < 0 || toIndex >= len(this.RuleSets) {
		return
	}
	if fromIndex == toIndex {
		return
	}

	location := this.RuleSets[fromIndex]
	result := []*RuleSet{}
	for i := 0; i < len(this.RuleSets); i++ {
		if i == fromIndex {
			continue
		}
		if fromIndex > toIndex && i == toIndex {
			result = append(result, location)
		}
		result = append(result, this.RuleSets[i])
		if fromIndex < toIndex && i == toIndex {
			result = append(result, location)
		}
	}

	this.RuleSets = result
}
