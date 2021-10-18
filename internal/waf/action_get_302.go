package waf

import (
	"github.com/TeaOSLab/EdgeNode/internal/remotelogs"
	"github.com/TeaOSLab/EdgeNode/internal/utils"
	"github.com/TeaOSLab/EdgeNode/internal/waf/requests"
	"github.com/iwind/TeaGo/maps"
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

func (this *Get302Action) Perform(waf *WAF, group *RuleGroup, set *RuleSet, request requests.Request, writer http.ResponseWriter) (allow bool) {
	// 仅限于Get
	if request.WAFRaw().Method != http.MethodGet {
		return true
	}

	// 是否已经在白名单中
	if SharedIPWhiteList.Contains("set:"+set.Id, this.Scope, request.WAFServerId(), request.WAFRemoteIP()) {
		return true
	}

	var m = maps.Map{
		"url":       request.WAFRaw().URL.String(),
		"timestamp": time.Now().Unix(),
		"life":      this.Life,
		"scope":     this.Scope,
		"setId":     set.Id,
	}
	info, err := utils.SimpleEncryptMap(m)
	if err != nil {
		remotelogs.Error("WAF_GET_302_ACTION", "encode info failed: "+err.Error())
		return true
	}

	http.Redirect(writer, request.WAFRaw(), Get302Path+"?info="+url.QueryEscape(info), http.StatusFound)

	// 关闭连接
	if request.WAFRaw().ProtoMajor == 1 {
		_ = this.CloseConn(writer)
	}

	return true
}
