package waf

import (
	"github.com/TeaOSLab/EdgeNode/internal/remotelogs"
	"github.com/TeaOSLab/EdgeNode/internal/utils"
	"github.com/TeaOSLab/EdgeNode/internal/waf/requests"
	"github.com/iwind/TeaGo/maps"
	"github.com/iwind/TeaGo/types"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	CaptchaSeconds = 600 // 10 minutes
	CaptchaPath    = "/WAF/VERIFY/CAPTCHA"
)

type CaptchaAction struct {
	BaseAction

	Life              int32 `yaml:"life" json:"life"`
	MaxFails          int   `yaml:"maxFails" json:"maxFails"`                   // 最大失败次数
	FailBlockTimeout  int   `yaml:"failBlockTimeout" json:"failBlockTimeout"`   // 失败拦截时间
	FailBlockScopeAll bool  `yaml:"failBlockScopeAll" json:"failBlockScopeAll"` // 是否全局有效

	CountLetters int8 `yaml:"countLetters" json:"countLetters"`

	UIIsOn          bool   `yaml:"uiIsOn" json:"uiIsOn"`                   // 是否使用自定义UI
	UITitle         string `yaml:"uiTitle" json:"uiTitle"`                 // 消息标题
	UIPrompt        string `yaml:"uiPrompt" json:"uiPrompt"`               // 消息提示
	UIButtonTitle   string `yaml:"uiButtonTitle" json:"uiButtonTitle"`     // 按钮标题
	UIShowRequestId bool   `yaml:"uiShowRequestId" json:"uiShowRequestId"` // 是否显示请求ID
	UICss           string `yaml:"uiCss" json:"uiCss"`                     // CSS样式
	UIFooter        string `yaml:"uiFooter" json:"uiFooter"`               // 页脚
	UIBody          string `yaml:"uiBody" json:"uiBody"`                   // 内容轮廓

	Lang           string `yaml:"lang" json:"lang"`                     // 语言，zh-CN, en-US ...
	AddToWhiteList bool   `yaml:"addToWhiteList" json:"addToWhiteList"` // 是否加入到白名单
	Scope          string `yaml:"scope" json:"scope"`
}

func (this *CaptchaAction) Init(waf *WAF) error {
	if waf.DefaultCaptchaAction != nil {
		if this.Life <= 0 {
			this.Life = waf.DefaultCaptchaAction.Life
		}
		if this.MaxFails <= 0 {
			this.MaxFails = waf.DefaultCaptchaAction.MaxFails
		}
		if this.FailBlockTimeout <= 0 {
			this.FailBlockTimeout = waf.DefaultCaptchaAction.FailBlockTimeout
		}
		this.FailBlockScopeAll = waf.DefaultCaptchaAction.FailBlockScopeAll

		if this.CountLetters <= 0 {
			this.CountLetters = waf.DefaultCaptchaAction.CountLetters
		}

		this.UIIsOn = waf.DefaultCaptchaAction.UIIsOn
		if len(this.UITitle) == 0 {
			this.UITitle = waf.DefaultCaptchaAction.UITitle
		}
		if len(this.UIPrompt) == 0 {
			this.UIPrompt = waf.DefaultCaptchaAction.UIPrompt
		}
		if len(this.UIButtonTitle) == 0 {
			this.UIButtonTitle = waf.DefaultCaptchaAction.UIButtonTitle
		}
		this.UIShowRequestId = waf.DefaultCaptchaAction.UIShowRequestId
		if len(this.UICss) == 0 {
			this.UICss = waf.DefaultCaptchaAction.UICss
		}
		if len(this.UIFooter) == 0 {
			this.UIFooter = waf.DefaultCaptchaAction.UIFooter
		}
		if len(this.UIBody) == 0 {
			this.UIBody = waf.DefaultCaptchaAction.UIBody
		}
		if len(this.Lang) == 0 {
			this.Lang = waf.DefaultCaptchaAction.Lang
		}
	}

	return nil
}

func (this *CaptchaAction) Code() string {
	return ActionCaptcha
}

func (this *CaptchaAction) IsAttack() bool {
	return false
}

func (this *CaptchaAction) WillChange() bool {
	return true
}

func (this *CaptchaAction) Perform(waf *WAF, group *RuleGroup, set *RuleSet, req requests.Request, writer http.ResponseWriter) (continueRequest bool, goNextSet bool) {
	// 是否在白名单中
	if SharedIPWhiteList.Contains("set:"+types.String(set.Id), this.Scope, req.WAFServerId(), req.WAFRemoteIP()) {
		return true, false
	}

	var refURL = req.WAFRaw().URL.String()

	// 覆盖配置
	if strings.HasPrefix(refURL, CaptchaPath) {
		info := req.WAFRaw().URL.Query().Get("info")
		if len(info) > 0 {
			m, err := utils.SimpleDecryptMap(info)
			if err == nil && m != nil {
				refURL = m.GetString("url")
			}
		}
	}

	var captchaConfig = maps.Map{
		"actionId":  this.ActionId(),
		"timestamp": time.Now().Unix(),
		"url":       refURL,
		"policyId":  waf.Id,
		"groupId":   group.Id,
		"setId":     set.Id,
	}
	info, err := utils.SimpleEncryptMap(captchaConfig)
	if err != nil {
		remotelogs.Error("WAF_CAPTCHA_ACTION", "encode captcha config failed: "+err.Error())
		return true, false
	}

	// 占用一次失败次数
	CaptchaIncreaseFails(req, this, waf.Id, group.Id, set.Id, CaptchaPageCodeInit)

	req.ProcessResponseHeaders(writer.Header(), http.StatusTemporaryRedirect)
	http.Redirect(writer, req.WAFRaw(), CaptchaPath+"?info="+url.QueryEscape(info), http.StatusTemporaryRedirect)

	return false, false
}
