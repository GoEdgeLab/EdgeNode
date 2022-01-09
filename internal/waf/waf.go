package waf

import (
	"errors"
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs/firewallconfigs"
	teaconst "github.com/TeaOSLab/EdgeNode/internal/const"
	"github.com/TeaOSLab/EdgeNode/internal/waf/checkpoints"
	"github.com/TeaOSLab/EdgeNode/internal/waf/requests"
	"github.com/iwind/TeaGo/Tea"
	"github.com/iwind/TeaGo/files"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"net/http"
	"reflect"
)

type WAF struct {
	Id               int64                        `yaml:"id" json:"id"`
	IsOn             bool                         `yaml:"isOn" json:"isOn"`
	Name             string                       `yaml:"name" json:"name"`
	Inbound          []*RuleGroup                 `yaml:"inbound" json:"inbound"`
	Outbound         []*RuleGroup                 `yaml:"outbound" json:"outbound"`
	CreatedVersion   string                       `yaml:"createdVersion" json:"createdVersion"`
	Mode             firewallconfigs.FirewallMode `yaml:"mode" json:"mode"`
	UseLocalFirewall bool                         `yaml:"useLocalFirewall" json:"useLocalFirewall"`

	DefaultBlockAction *BlockAction

	hasInboundRules  bool
	hasOutboundRules bool

	checkpointsMap map[string]checkpoints.CheckpointInterface // prefix => checkpoint
}

func NewWAF() *WAF {
	return &WAF{
		IsOn: true,
	}
}

func NewWAFFromFile(path string) (waf *WAF, err error) {
	if len(path) == 0 {
		return nil, errors.New("'path' should not be empty")
	}
	file := files.NewFile(path)
	if !file.Exists() {
		return nil, errors.New("'" + path + "' not exist")
	}

	reader, err := file.Reader()
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = reader.Close()
	}()

	waf = &WAF{}
	err = reader.ReadYAML(waf)
	if err != nil {
		return nil, err
	}
	return waf, nil
}

func (this *WAF) Init() (resultErrors []error) {
	// checkpoint
	this.checkpointsMap = map[string]checkpoints.CheckpointInterface{}
	for _, def := range checkpoints.AllCheckpoints {
		instance := reflect.New(reflect.Indirect(reflect.ValueOf(def.Instance)).Type()).Interface().(checkpoints.CheckpointInterface)
		instance.Init()
		this.checkpointsMap[def.Prefix] = instance
	}

	// rules
	this.hasInboundRules = len(this.Inbound) > 0
	this.hasOutboundRules = len(this.Outbound) > 0

	if this.hasInboundRules {
		for _, group := range this.Inbound {
			// finder
			for _, set := range group.RuleSets {
				for _, rule := range set.Rules {
					rule.SetCheckpointFinder(this.FindCheckpointInstance)
				}
			}

			err := group.Init(this)
			if err != nil {
				// 这里我们不阻止其他规则正常加入
				resultErrors = append(resultErrors, err)
			}
		}
	}

	if this.hasOutboundRules {
		for _, group := range this.Outbound {
			// finder
			for _, set := range group.RuleSets {
				for _, rule := range set.Rules {
					rule.SetCheckpointFinder(this.FindCheckpointInstance)
				}
			}

			err := group.Init(this)
			if err != nil {
				// 这里我们不阻止其他规则正常加入
				resultErrors = append(resultErrors, err)
			}
		}
	}

	return nil
}

func (this *WAF) AddRuleGroup(ruleGroup *RuleGroup) {
	if ruleGroup.IsInbound {
		this.Inbound = append(this.Inbound, ruleGroup)
	} else {
		this.Outbound = append(this.Outbound, ruleGroup)
	}
}

func (this *WAF) RemoveRuleGroup(ruleGroupId int64) {
	{
		result := []*RuleGroup{}
		for _, group := range this.Inbound {
			if group.Id == ruleGroupId {
				continue
			}
			result = append(result, group)
		}
		this.Inbound = result
	}

	{
		result := []*RuleGroup{}
		for _, group := range this.Outbound {
			if group.Id == ruleGroupId {
				continue
			}
			result = append(result, group)
		}
		this.Outbound = result
	}
}

func (this *WAF) FindRuleGroup(ruleGroupId int64) *RuleGroup {
	for _, group := range this.Inbound {
		if group.Id == ruleGroupId {
			return group
		}
	}
	for _, group := range this.Outbound {
		if group.Id == ruleGroupId {
			return group
		}
	}
	return nil
}

func (this *WAF) FindRuleGroupWithCode(ruleGroupCode string) *RuleGroup {
	if len(ruleGroupCode) == 0 {
		return nil
	}
	for _, group := range this.Inbound {
		if group.Code == ruleGroupCode {
			return group
		}
	}
	for _, group := range this.Outbound {
		if group.Code == ruleGroupCode {
			return group
		}
	}
	return nil
}

func (this *WAF) MoveInboundRuleGroup(fromIndex int, toIndex int) {
	if fromIndex < 0 || fromIndex >= len(this.Inbound) {
		return
	}
	if toIndex < 0 || toIndex >= len(this.Inbound) {
		return
	}
	if fromIndex == toIndex {
		return
	}

	group := this.Inbound[fromIndex]
	result := []*RuleGroup{}
	for i := 0; i < len(this.Inbound); i++ {
		if i == fromIndex {
			continue
		}
		if fromIndex > toIndex && i == toIndex {
			result = append(result, group)
		}
		result = append(result, this.Inbound[i])
		if fromIndex < toIndex && i == toIndex {
			result = append(result, group)
		}
	}

	this.Inbound = result
}

func (this *WAF) MoveOutboundRuleGroup(fromIndex int, toIndex int) {
	if fromIndex < 0 || fromIndex >= len(this.Outbound) {
		return
	}
	if toIndex < 0 || toIndex >= len(this.Outbound) {
		return
	}
	if fromIndex == toIndex {
		return
	}

	group := this.Outbound[fromIndex]
	result := []*RuleGroup{}
	for i := 0; i < len(this.Outbound); i++ {
		if i == fromIndex {
			continue
		}
		if fromIndex > toIndex && i == toIndex {
			result = append(result, group)
		}
		result = append(result, this.Outbound[i])
		if fromIndex < toIndex && i == toIndex {
			result = append(result, group)
		}
	}

	this.Outbound = result
}

func (this *WAF) MatchRequest(req requests.Request, writer http.ResponseWriter) (goNext bool, group *RuleGroup, set *RuleSet, err error) {
	if !this.hasInboundRules {
		return true, nil, nil, nil
	}

	// validate captcha
	var rawPath = req.WAFRaw().URL.Path
	if rawPath == CaptchaPath {
		captchaValidator.Run(req, writer)
		return
	}

	// Get 302验证
	if rawPath == Get302Path {
		get302Validator.Run(req, writer)
		return
	}

	// match rules
	for _, group := range this.Inbound {
		if !group.IsOn {
			continue
		}
		b, set, err := group.MatchRequest(req)
		if err != nil {
			return true, nil, nil, err
		}
		if b {
			goNext := set.PerformActions(this, group, req, writer)
			return goNext, group, set, nil
		}
	}
	return true, nil, nil, nil
}

func (this *WAF) MatchResponse(req requests.Request, rawResp *http.Response, writer http.ResponseWriter) (goNext bool, group *RuleGroup, set *RuleSet, err error) {
	if !this.hasOutboundRules {
		return true, nil, nil, nil
	}
	resp := requests.NewResponse(rawResp)
	for _, group := range this.Outbound {
		if !group.IsOn {
			continue
		}
		b, set, err := group.MatchResponse(req, resp)
		if err != nil {
			return true, nil, nil, err
		}
		if b {
			goNext := set.PerformActions(this, group, req, writer)
			return goNext, group, set, nil
		}
	}
	return true, nil, nil, nil
}

// Save save to file path
func (this *WAF) Save(path string) error {
	if len(path) == 0 {
		return errors.New("path should not be empty")
	}
	if len(this.CreatedVersion) == 0 {
		this.CreatedVersion = teaconst.Version
	}
	data, err := yaml.Marshal(this)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(path, data, 0644)
}

func (this *WAF) ContainsGroupCode(code string) bool {
	if len(code) == 0 {
		return false
	}
	for _, group := range this.Inbound {
		if group.Code == code {
			return true
		}
	}
	for _, group := range this.Outbound {
		if group.Code == code {
			return true
		}
	}
	return false
}

func (this *WAF) Copy() *WAF {
	waf := &WAF{
		Id:       this.Id,
		IsOn:     this.IsOn,
		Name:     this.Name,
		Inbound:  this.Inbound,
		Outbound: this.Outbound,
	}
	return waf
}

func (this *WAF) CountInboundRuleSets() int {
	count := 0
	for _, group := range this.Inbound {
		count += len(group.RuleSets)
	}
	return count
}

func (this *WAF) CountOutboundRuleSets() int {
	count := 0
	for _, group := range this.Outbound {
		count += len(group.RuleSets)
	}
	return count
}

func (this *WAF) FindCheckpointInstance(prefix string) checkpoints.CheckpointInterface {
	instance, ok := this.checkpointsMap[prefix]
	if ok {
		return instance
	}
	return nil
}

// Start start
func (this *WAF) Start() {
	for _, checkpoint := range this.checkpointsMap {
		checkpoint.Start()
	}
}

// Stop call stop() when the waf was deleted
func (this *WAF) Stop() {
	for _, checkpoint := range this.checkpointsMap {
		checkpoint.Stop()
	}
}

// MergeTemplate merge with template
func (this *WAF) MergeTemplate() (changedItems []string) {
	changedItems = []string{}

	// compare versions
	if !Tea.IsTesting() && this.CreatedVersion == teaconst.Version {
		return
	}
	this.CreatedVersion = teaconst.Version

	template := Template()
	groups := []*RuleGroup{}
	groups = append(groups, template.Inbound...)
	groups = append(groups, template.Outbound...)

	var newGroupId int64 = 1_000_000_000

	for _, group := range groups {
		oldGroup := this.FindRuleGroupWithCode(group.Code)
		if oldGroup == nil {
			newGroupId++
			group.Id = newGroupId
			this.AddRuleGroup(group)
			changedItems = append(changedItems, "+group "+group.Name)
			continue
		}

		// check rule sets
		for _, set := range group.RuleSets {
			oldSet := oldGroup.FindRuleSetWithCode(set.Code)
			if oldSet == nil {
				oldGroup.AddRuleSet(set)
				changedItems = append(changedItems, "+group "+group.Name+" rule set:"+set.Name)
			} else if len(oldSet.Rules) < len(set.Rules) {
				oldSet.Rules = set.Rules
				changedItems = append(changedItems, "*group "+group.Name+" rule set:"+set.Name)
			}
		}
	}
	return
}
