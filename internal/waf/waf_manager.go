package waf

import (
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs/firewallconfigs"
	"github.com/TeaOSLab/EdgeNode/internal/errors"
	"github.com/TeaOSLab/EdgeNode/internal/remotelogs"
	"strconv"
	"sync"
)

var SharedWAFManager = NewWAFManager()

// WAFManager WAF管理器
type WAFManager struct {
	mapping map[int64]*WAF // policyId => WAF
	locker  sync.RWMutex
}

// NewWAFManager 获取新对象
func NewWAFManager() *WAFManager {
	return &WAFManager{
		mapping: map[int64]*WAF{},
	}
}

// UpdatePolicies 更新策略
func (this *WAFManager) UpdatePolicies(policies []*firewallconfigs.HTTPFirewallPolicy) {
	this.locker.Lock()
	defer this.locker.Unlock()

	m := map[int64]*WAF{}
	for _, p := range policies {
		w, err := this.ConvertWAF(p)
		if w != nil {
			m[p.Id] = w
		}
		if err != nil {
			remotelogs.Error("WAF", "initialize policy '"+strconv.FormatInt(p.Id, 10)+"' failed: "+err.Error())
			continue
		}
	}
	this.mapping = m
}

// FindWAF 查找WAF
func (this *WAFManager) FindWAF(policyId int64) *WAF {
	this.locker.RLock()
	var w = this.mapping[policyId]
	this.locker.RUnlock()
	return w
}

// ConvertWAF 将Policy转换为WAF
func (this *WAFManager) ConvertWAF(policy *firewallconfigs.HTTPFirewallPolicy) (*WAF, error) {
	if policy == nil {
		return nil, errors.New("policy should not be nil")
	}
	if len(policy.Mode) == 0 {
		policy.Mode = firewallconfigs.FirewallModeDefend
	}
	var w = &WAF{
		Id:               policy.Id,
		IsOn:             policy.IsOn,
		Name:             policy.Name,
		Mode:             policy.Mode,
		UseLocalFirewall: policy.UseLocalFirewall,
		SYNFlood:         policy.SYNFlood,
	}

	// inbound
	if policy.Inbound != nil && policy.Inbound.IsOn {
		for _, group := range policy.Inbound.Groups {
			g := &RuleGroup{
				Id:          group.Id,
				IsOn:        group.IsOn,
				Name:        group.Name,
				Description: group.Description,
				Code:        group.Code,
				IsInbound:   true,
			}

			// rule sets
			for _, set := range group.Sets {
				s := &RuleSet{
					Id:          set.Id,
					Code:        set.Code,
					IsOn:        set.IsOn,
					Name:        set.Name,
					Description: set.Description,
					Connector:   set.Connector,
					IgnoreLocal: set.IgnoreLocal,
				}
				for _, a := range set.Actions {
					s.AddAction(a.Code, a.Options)
				}

				// rules
				for _, rule := range set.Rules {
					r := &Rule{
						Id:                rule.Id,
						Description:       rule.Description,
						Param:             rule.Param,
						ParamFilters:      []*ParamFilter{},
						Operator:          rule.Operator,
						Value:             rule.Value,
						IsCaseInsensitive: rule.IsCaseInsensitive,
						CheckpointOptions: rule.CheckpointOptions,
					}

					for _, paramFilter := range rule.ParamFilters {
						r.ParamFilters = append(r.ParamFilters, &ParamFilter{
							Code:    paramFilter.Code,
							Options: paramFilter.Options,
						})
					}

					s.Rules = append(s.Rules, r)
				}

				g.RuleSets = append(g.RuleSets, s)
			}

			w.Inbound = append(w.Inbound, g)
		}
	}

	// outbound
	if policy.Outbound != nil && policy.Outbound.IsOn {
		for _, group := range policy.Outbound.Groups {
			g := &RuleGroup{
				Id:          group.Id,
				IsOn:        group.IsOn,
				Name:        group.Name,
				Description: group.Description,
				Code:        group.Code,
				IsInbound:   true,
			}

			// rule sets
			for _, set := range group.Sets {
				s := &RuleSet{
					Id:          set.Id,
					Code:        set.Code,
					IsOn:        set.IsOn,
					Name:        set.Name,
					Description: set.Description,
					Connector:   set.Connector,
					IgnoreLocal: set.IgnoreLocal,
				}

				for _, a := range set.Actions {
					s.AddAction(a.Code, a.Options)
				}

				// rules
				for _, rule := range set.Rules {
					r := &Rule{
						Id:                rule.Id,
						Description:       rule.Description,
						Param:             rule.Param,
						Operator:          rule.Operator,
						Value:             rule.Value,
						IsCaseInsensitive: rule.IsCaseInsensitive,
						CheckpointOptions: rule.CheckpointOptions,
					}
					s.Rules = append(s.Rules, r)
				}

				g.RuleSets = append(g.RuleSets, s)
			}

			w.Outbound = append(w.Outbound, g)
		}
	}

	// block action
	if policy.BlockOptions != nil {
		w.DefaultBlockAction = &BlockAction{
			StatusCode: policy.BlockOptions.StatusCode,
			Body:       policy.BlockOptions.Body,
			URL:        policy.BlockOptions.URL,
			Timeout:    policy.BlockOptions.Timeout,
			TimeoutMax: policy.BlockOptions.TimeoutMax,
		}
	}

	// page action
	if policy.PageOptions != nil {
		w.DefaultPageAction = &PageAction{
			Status: policy.PageOptions.Status,
			Body:   policy.PageOptions.Body,
		}
	}

	// captcha action
	if policy.CaptchaOptions != nil {
		w.DefaultCaptchaAction = &CaptchaAction{
			Life:              policy.CaptchaOptions.Life,
			MaxFails:          policy.CaptchaOptions.MaxFails,
			FailBlockTimeout:  policy.CaptchaOptions.FailBlockTimeout,
			FailBlockScopeAll: policy.CaptchaOptions.FailBlockScopeAll,
			CountLetters:      policy.CaptchaOptions.CountLetters,
			CaptchaType:       policy.CaptchaOptions.CaptchaType,
			UIIsOn:            policy.CaptchaOptions.UIIsOn,
			UITitle:           policy.CaptchaOptions.UITitle,
			UIPrompt:          policy.CaptchaOptions.UIPrompt,
			UIButtonTitle:     policy.CaptchaOptions.UIButtonTitle,
			UIShowRequestId:   policy.CaptchaOptions.UIShowRequestId,
			UICss:             policy.CaptchaOptions.UICss,
			UIFooter:          policy.CaptchaOptions.UIFooter,
			UIBody:            policy.CaptchaOptions.UIBody,
			Lang:              policy.CaptchaOptions.Lang,
			GeeTestConfig:     &policy.CaptchaOptions.GeeTestConfig,
		}
	}

	errorList := w.Init()
	if len(errorList) > 0 {
		return w, errorList[0]
	}

	return w, nil
}
