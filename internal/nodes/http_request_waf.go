package nodes

import (
	"bytes"
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs/firewallconfigs"
	"github.com/TeaOSLab/EdgeNode/internal/iplibrary"
	"github.com/TeaOSLab/EdgeNode/internal/remotelogs"
	"github.com/TeaOSLab/EdgeNode/internal/stats"
	"github.com/TeaOSLab/EdgeNode/internal/waf"
	"github.com/iwind/TeaGo/lists"
	"github.com/iwind/TeaGo/types"
	"io"
	"io/ioutil"
	"net/http"
)

// 调用WAF
func (this *HTTPRequest) doWAFRequest() (blocked bool) {
	// 当前连接是否已关闭
	var conn = this.RawReq.Context().Value(HTTPConnContextKey)
	if conn != nil {
		trafficConn, ok := conn.(*TrafficConn)
		if ok && trafficConn.IsClosed() {
			return true
		}
	}

	// 当前服务的独立设置
	if this.web.FirewallPolicy != nil && this.web.FirewallPolicy.IsOn {
		blocked, breakChecking := this.checkWAFRequest(this.web.FirewallPolicy)
		if blocked {
			return true
		}
		if breakChecking {
			return false
		}
	}

	// 公用的防火墙设置
	if this.Server.HTTPFirewallPolicy != nil && this.Server.HTTPFirewallPolicy.IsOn {
		blocked, breakChecking := this.checkWAFRequest(this.Server.HTTPFirewallPolicy)
		if blocked {
			return true
		}
		if breakChecking {
			return false
		}
	}

	return
}

func (this *HTTPRequest) checkWAFRequest(firewallPolicy *firewallconfigs.HTTPFirewallPolicy) (blocked bool, breakChecking bool) {
	// 检查配置是否为空
	if firewallPolicy == nil || !firewallPolicy.IsOn || firewallPolicy.Inbound == nil || !firewallPolicy.Inbound.IsOn {
		return
	}

	// 检查IP白名单
	remoteAddrs := this.requestRemoteAddrs()
	inbound := firewallPolicy.Inbound
	if inbound == nil {
		return
	}
	for _, ref := range inbound.AllAllowListRefs() {
		if ref.IsOn && ref.ListId > 0 {
			list := iplibrary.SharedIPListManager.FindList(ref.ListId)
			if list != nil {
				found, _ := list.ContainsIPStrings(remoteAddrs)
				if found {
					breakChecking = true
					return
				}
			}
		}
	}

	// 检查IP黑名单
	for _, ref := range inbound.AllDenyListRefs() {
		if ref.IsOn && ref.ListId > 0 {
			list := iplibrary.SharedIPListManager.FindList(ref.ListId)
			if list != nil {
				found, item := list.ContainsIPStrings(remoteAddrs)
				if found {
					// 触发事件
					if item != nil && len(item.EventLevel) > 0 {
						actions := iplibrary.SharedActionManager.FindEventActions(item.EventLevel)
						for _, action := range actions {
							goNext, err := action.DoHTTP(this.RawReq, this.RawWriter)
							if err != nil {
								remotelogs.Error("HTTP_REQUEST_WAF", "do action '"+err.Error()+"' failed: "+err.Error())
								return true, false
							}
							if !goNext {
								this.disableLog = true
								return true, false
							}
						}
					}

					// TODO 需要记录日志信息

					this.writer.WriteHeader(http.StatusForbidden)
					this.writer.Close()

					// 停止日志
					this.disableLog = true

					return true, false
				}
			}
		}
	}

	// 检查地区封禁
	if iplibrary.SharedLibrary != nil {
		if firewallPolicy.Inbound.Region != nil && firewallPolicy.Inbound.Region.IsOn {
			regionConfig := firewallPolicy.Inbound.Region
			if regionConfig.IsNotEmpty() {
				for _, remoteAddr := range remoteAddrs {
					result, err := iplibrary.SharedLibrary.Lookup(remoteAddr)
					if err != nil {
						remotelogs.Error("HTTP_REQUEST_WAF", "iplibrary lookup failed: "+err.Error())
					} else if result != nil {
						// 检查国家级别封禁
						if len(regionConfig.DenyCountryIds) > 0 && len(result.Country) > 0 {
							countryId := iplibrary.SharedCountryManager.Lookup(result.Country)
							if countryId > 0 && lists.ContainsInt64(regionConfig.DenyCountryIds, countryId) {
								// TODO 可以配置对封禁的处理方式等
								// TODO 需要记录日志信息
								this.writer.WriteHeader(http.StatusForbidden)
								this.writer.Close()

								// 停止日志
								this.disableLog = true

								return true, false
							}
						}

						// 检查省份封禁
						if len(regionConfig.DenyProvinceIds) > 0 && len(result.Province) > 0 {
							provinceId := iplibrary.SharedProvinceManager.Lookup(result.Province)
							if provinceId > 0 && lists.ContainsInt64(regionConfig.DenyProvinceIds, provinceId) {
								// TODO 可以配置对封禁的处理方式等
								// TODO 需要记录日志信息
								this.writer.WriteHeader(http.StatusForbidden)
								this.writer.Close()

								// 停止日志
								this.disableLog = true

								return true, false
							}
						}
					}
				}
			}
		}
	}

	// 规则测试
	w := sharedWAFManager.FindWAF(firewallPolicy.Id)
	if w == nil {
		return
	}

	w.OnAction(func(action waf.ActionInterface) (goNext bool) {
		switch action.Code() {
		case waf.ActionTag:
			this.tags = action.(*waf.TagAction).Tags
		}
		return true
	})

	goNext, ruleGroup, ruleSet, err := w.MatchRequest(this, this.writer)
	if err != nil {
		remotelogs.Error("HTTP_REQUEST_WAF", this.rawURI+": "+err.Error())
		return
	}

	if ruleSet != nil {
		if ruleSet.HasSpecialActions() {
			this.firewallPolicyId = firewallPolicy.Id
			this.firewallRuleGroupId = types.Int64(ruleGroup.Id)
			this.firewallRuleSetId = types.Int64(ruleSet.Id)

			if ruleSet.HasAttackActions() {
				this.isAttack = true
			}

			// 添加统计
			stats.SharedHTTPRequestStatManager.AddFirewallRuleGroupId(this.Server.Id, this.firewallRuleGroupId, ruleSet.Actions)
		}

		this.firewallActions = ruleSet.ActionCodes()
	}

	return !goNext, false
}

// call response waf
func (this *HTTPRequest) doWAFResponse(resp *http.Response) (blocked bool) {
	// 当前服务的独立设置
	if this.web.FirewallPolicy != nil && this.web.FirewallPolicy.IsOn {
		blocked := this.checkWAFResponse(this.web.FirewallPolicy, resp)
		if blocked {
			return true
		}
	}

	// 公用的防火墙设置
	if this.Server.HTTPFirewallPolicy != nil && this.Server.HTTPFirewallPolicy.IsOn {
		blocked := this.checkWAFResponse(this.Server.HTTPFirewallPolicy, resp)
		if blocked {
			return true
		}
	}
	return
}

func (this *HTTPRequest) checkWAFResponse(firewallPolicy *firewallconfigs.HTTPFirewallPolicy, resp *http.Response) (blocked bool) {
	if firewallPolicy == nil || !firewallPolicy.IsOn || !firewallPolicy.Outbound.IsOn {
		return
	}

	w := sharedWAFManager.FindWAF(firewallPolicy.Id)
	if w == nil {
		return
	}

	w.OnAction(func(action waf.ActionInterface) (goNext bool) {
		switch action.Code() {
		case waf.ActionTag:
			this.tags = action.(*waf.TagAction).Tags
		}
		return true
	})

	goNext, ruleGroup, ruleSet, err := w.MatchResponse(this, resp, this.writer)
	if err != nil {
		remotelogs.Error("HTTP_REQUEST_WAF", this.rawURI+": "+err.Error())
		return
	}

	if ruleSet != nil {
		if ruleSet.HasSpecialActions() {
			this.firewallPolicyId = firewallPolicy.Id
			this.firewallRuleGroupId = types.Int64(ruleGroup.Id)
			this.firewallRuleSetId = types.Int64(ruleSet.Id)

			if ruleSet.HasAttackActions() {
				this.isAttack = true
			}

			// 添加统计
			stats.SharedHTTPRequestStatManager.AddFirewallRuleGroupId(this.Server.Id, this.firewallRuleGroupId, ruleSet.Actions)
		}

		this.firewallActions = ruleSet.ActionCodes()
	}

	return !goNext
}

// WAFRaw 原始请求
func (this *HTTPRequest) WAFRaw() *http.Request {
	return this.RawReq
}

// WAFRemoteIP 客户端IP
func (this *HTTPRequest) WAFRemoteIP() string {
	return this.requestRemoteAddr()
}

// WAFGetCacheBody 获取缓存中的Body
func (this *HTTPRequest) WAFGetCacheBody() []byte {
	return this.bodyData
}

// WAFSetCacheBody 设置Body
func (this *HTTPRequest) WAFSetCacheBody(body []byte) {
	this.bodyData = body
}

// WAFReadBody 读取Body
func (this *HTTPRequest) WAFReadBody(max int64) (data []byte, err error) {
	if this.RawReq.ContentLength > 0 {
		data, err = ioutil.ReadAll(io.LimitReader(this.RawReq.Body, max))
	}
	return
}

// WAFRestoreBody 恢复Body
func (this *HTTPRequest) WAFRestoreBody(data []byte) {
	if len(data) > 0 {
		rawReader := bytes.NewBuffer(data)
		buf := make([]byte, 1024)
		_, _ = io.CopyBuffer(rawReader, this.RawReq.Body, buf)
		this.RawReq.Body = ioutil.NopCloser(rawReader)
	}
}

// WAFServerId 服务ID
func (this *HTTPRequest) WAFServerId() int64 {
	return this.Server.Id
}
