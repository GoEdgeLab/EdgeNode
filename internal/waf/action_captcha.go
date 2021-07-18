package waf

import (
	"github.com/TeaOSLab/EdgeNode/internal/remotelogs"
	"github.com/TeaOSLab/EdgeNode/internal/utils"
	"github.com/TeaOSLab/EdgeNode/internal/waf/requests"
	"github.com/iwind/TeaGo/maps"
	stringutil "github.com/iwind/TeaGo/utils/string"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var captchaSalt = stringutil.Rand(32)

const (
	CaptchaSeconds = 600 // 10 minutes
	CaptchaPath    = "/WAF/VERIFY/CAPTCHA"
)

type CaptchaAction struct {
	Life           int32  `yaml:"life" json:"life"`
	Language       string `yaml:"language" json:"language"`             // 语言，zh-CN, en-US ...
	AddToWhiteList bool   `yaml:"addToWhiteList" json:"addToWhiteList"` // 是否加入到白名单
}

func (this *CaptchaAction) Init(waf *WAF) error {
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

func (this *CaptchaAction) Perform(waf *WAF, group *RuleGroup, set *RuleSet, request requests.Request, writer http.ResponseWriter) (allow bool) {
	// 是否在白名单中
	if SharedIPWhiteList.Contains("set:"+set.Id, request.WAFRemoteIP()) {
		return true
	}

	refURL := request.WAFRaw().URL.String()

	// 覆盖配置
	if strings.HasPrefix(refURL, CaptchaPath) {
		info := request.WAFRaw().URL.Query().Get("info")
		if len(info) > 0 {
			m, err := utils.SimpleDecryptMap(info)
			if err == nil && m != nil {
				refURL = m.GetString("url")
			}
		}
	}

	var captchaConfig = maps.Map{
		"action":    this,
		"timestamp": time.Now().Unix(),
		"url":       refURL,
		"setId":     set.Id,
	}
	info, err := utils.SimpleEncryptMap(captchaConfig)
	if err != nil {
		remotelogs.Error("WAF_CAPTCHA_ACTION", "encode captcha config failed: "+err.Error())
		return true
	}

	http.Redirect(writer, request.WAFRaw(), CaptchaPath+"?info="+url.QueryEscape(info), http.StatusTemporaryRedirect)

	return false
}
