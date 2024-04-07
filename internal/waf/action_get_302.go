package waf

import (
	"github.com/TeaOSLab/EdgeNode/internal/remotelogs"
	"github.com/TeaOSLab/EdgeNode/internal/utils"
	"github.com/TeaOSLab/EdgeNode/internal/waf/requests"
	"github.com/iwind/TeaGo/types"
	"net/http"
	"net/url"
	"time"
)

const (
	Get302Path = "/WAF/VERIFY/GET"
)

// Get302Action
// 原理：  origin url --> 302 verify url --> origin url
// TODO 将来支持meta refresh验证
type Get302Action struct {
	BaseAction

	Life  int32  `yaml:"life" json:"life"`
	Scope string `yaml:"scope" json:"scope"`
}

func (this *Get302Action) Init(waf *WAF) error {
	return nil
}

func (this *Get302Action) Code() string {
	return ActionGet302
}

func (this *Get302Action) IsAttack() bool {
	return false
}

func (this *Get302Action) WillChange() bool {
	return true
}

func (this *Get302Action) Perform(waf *WAF, group *RuleGroup, set *RuleSet, request requests.Request, writer http.ResponseWriter) PerformResult {
	// 仅限于Get
	if request.WAFRaw().Method != http.MethodGet {
		return PerformResult{
			ContinueRequest: true,
		}
	}

	// 是否已经在白名单中
	if SharedIPWhiteList.Contains("set:"+types.String(set.Id), this.Scope, request.WAFServerId(), request.WAFRemoteIP()) {
		return PerformResult{
			ContinueRequest: true,
		}
	}

	var m = InfoArg{
		URL:              request.WAFRaw().URL.String(),
		Timestamp:        time.Now().Unix(),
		Life:             this.Life,
		Scope:            this.Scope,
		PolicyId:         waf.Id,
		GroupId:          group.Id,
		SetId:            set.Id,
		UseLocalFirewall: false,
	}
	info, err := utils.SimpleEncryptObject(m)
	if err != nil {
		remotelogs.Error("WAF_GET_302_ACTION", "encode info failed: "+err.Error())
		return PerformResult{
			ContinueRequest: true,
		}
	}

	request.DisableStat()
	request.ProcessResponseHeaders(writer.Header(), http.StatusFound)
	http.Redirect(writer, request.WAFRaw(), Get302Path+"?info="+url.QueryEscape(info), http.StatusFound)

	flusher, ok := writer.(http.Flusher)
	if ok {
		flusher.Flush()
	}

	return PerformResult{}
}
