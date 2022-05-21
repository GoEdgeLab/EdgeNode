package waf

import (
	"bytes"
	"encoding/base64"
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs/firewallconfigs"
	"github.com/TeaOSLab/EdgeNode/internal/ttlcache"
	"github.com/TeaOSLab/EdgeNode/internal/utils"
	"github.com/TeaOSLab/EdgeNode/internal/waf/requests"
	"github.com/dchest/captcha"
	"github.com/iwind/TeaGo/logs"
	"github.com/iwind/TeaGo/types"
	"net/http"
	"strconv"
	"strings"
	"time"
)

var captchaValidator = NewCaptchaValidator()

type CaptchaValidator struct {
}

func NewCaptchaValidator() *CaptchaValidator {
	return &CaptchaValidator{}
}

func (this *CaptchaValidator) Run(req requests.Request, writer http.ResponseWriter) {
	var info = req.WAFRaw().URL.Query().Get("info")
	if len(info) == 0 {
		writer.WriteHeader(http.StatusBadRequest)
		_, _ = writer.Write([]byte("invalid request"))
		return
	}
	m, err := utils.SimpleDecryptMap(info)
	if err != nil {
		_, _ = writer.Write([]byte("invalid request"))
		return
	}

	var timestamp = m.GetInt64("timestamp")
	if timestamp < time.Now().Unix()-600 { // 10分钟之后信息过期
		http.Redirect(writer, req.WAFRaw(), m.GetString("url"), http.StatusTemporaryRedirect)
		return
	}

	var actionId = m.GetInt64("actionId")
	var setId = m.GetInt64("setId")
	var originURL = m.GetString("url")
	var policyId = m.GetInt64("policyId")
	var groupId = m.GetInt64("groupId")

	var waf = SharedWAFManager.FindWAF(policyId)
	if waf == nil {
		http.Redirect(writer, req.WAFRaw(), originURL, http.StatusTemporaryRedirect)
		return
	}
	var actionConfig = waf.FindAction(actionId)
	if actionConfig == nil {
		http.Redirect(writer, req.WAFRaw(), originURL, http.StatusTemporaryRedirect)
		return
	}
	captchaActionConfig, ok := actionConfig.(*CaptchaAction)
	if !ok {
		http.Redirect(writer, req.WAFRaw(), originURL, http.StatusTemporaryRedirect)
		return
	}

	if req.WAFRaw().Method == http.MethodPost && len(req.WAFRaw().FormValue("GOEDGE_WAF_CAPTCHA_ID")) > 0 {
		this.validate(captchaActionConfig, policyId, groupId, setId, originURL, req, writer)
	} else {
		// 增加计数
		this.IncreaseFails(req, captchaActionConfig, policyId, groupId, setId)
		this.show(captchaActionConfig, req, writer)
	}
}

func (this *CaptchaValidator) show(actionConfig *CaptchaAction, req requests.Request, writer http.ResponseWriter) {
	// show captcha
	var countLetters = 6
	if actionConfig.CountLetters > 0 && actionConfig.CountLetters <= 10 {
		countLetters = int(actionConfig.CountLetters)
	}
	var captchaId = captcha.NewLen(countLetters)
	var buf = bytes.NewBuffer([]byte{})
	err := captcha.WriteImage(buf, captchaId, 200, 100)
	if err != nil {
		logs.Error(err)
		return
	}

	var lang = actionConfig.Lang
	if len(lang) == 0 {
		var acceptLanguage = req.WAFRaw().Header.Get("Accept-Language")
		if len(acceptLanguage) > 0 {
			langIndex := strings.Index(acceptLanguage, ",")
			if langIndex > 0 {
				lang = acceptLanguage[:langIndex]
			}
		}
	}
	if len(lang) == 0 {
		lang = "en-US"
	}

	var msgTitle = ""
	var msgPrompt = ""
	var msgButtonTitle = ""
	var msgRequestId = ""

	switch lang {
	case "en-US":
		msgTitle = "Verify Yourself"
		msgPrompt = "Input verify code above:"
		msgButtonTitle = "Verify Yourself"
		msgRequestId = "Request ID"
	case "zh-CN":
		msgTitle = "身份验证"
		msgPrompt = "请输入上面的验证码"
		msgButtonTitle = "提交验证"
		msgRequestId = "请求ID"
	default:
		msgTitle = "Verify Yourself"
		msgPrompt = "Input verify code above:"
		msgButtonTitle = "Verify Yourself"
		msgRequestId = "Request ID"
	}

	var msgCss = ""
	var requestIdBox = `<address>` + msgRequestId + `: ` + req.Format("${requestId}") + `</address>`
	var msgFooter = ""
	var body = `<form method="POST">
	<input type="hidden" name="GOEDGE_WAF_CAPTCHA_ID" value="` + captchaId + `"/>
	<div class="ui-image">
		<img src="data:image/png;base64, ` + base64.StdEncoding.EncodeToString(buf.Bytes()) + `"/>` + `
	</div>
	<div class="ui-input">
		<p>` + msgPrompt + `</p>
		<input type="text" name="GOEDGE_WAF_CAPTCHA_CODE" id="GOEDGE_WAF_CAPTCHA_CODE" maxlength="6" autocomplete="off" z-index="1" class="input"/>
	</div>
	<div class="ui-button">
		<button type="submit" style="line-height:24px;margin-top:10px">` + msgButtonTitle + `</button>
	</div>
</form>
` + requestIdBox + `
` + msgFooter + ``

	// 默认设置
	if actionConfig.UIIsOn {
		if len(actionConfig.UITitle) > 0 {
			msgTitle = actionConfig.UITitle
		}
		if len(actionConfig.UIPrompt) > 0 {
			msgPrompt = actionConfig.UIPrompt
		}
		if len(actionConfig.UIButtonTitle) > 0 {
			msgButtonTitle = actionConfig.UIButtonTitle
		}
		if len(actionConfig.UICss) > 0 {
			msgCss = actionConfig.UICss
		}
		if !actionConfig.UIShowRequestId {
			requestIdBox = ""
		}
		if len(actionConfig.UIFooter) > 0 {
			msgFooter = actionConfig.UIFooter
		}
		if len(actionConfig.UIBody) > 0 {
			var index = strings.Index(actionConfig.UIBody, "${body}")
			if index < 0 {
				body = actionConfig.UIBody + body
			} else {
				body = actionConfig.UIBody[:index] + body + actionConfig.UIBody[index+7:] // 7是"${body}"的长度
			}
		}
	}

	writer.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = writer.Write([]byte(`<!DOCTYPE html>
<html>
<head>
	<title>` + msgTitle + `</title>
	<meta name="viewport" content="width=device-width, initial-scale=1, user-scalable=0">
	<meta charset="UTF-8"/>
	<script type="text/javascript">
	if (window.addEventListener != null) {
		window.addEventListener("load", function () {
			document.getElementById("GOEDGE_WAF_CAPTCHA_CODE").focus()
		})
	}
	</script>
	<style type="text/css">
	form { width: 20em; margin: 0 auto; text-align: center; }
	.input { font-size:16px;line-height:24px; letter-spacing: 15px; padding-left: 10px; width: 140px; }
	address { margin-top: 1em; padding-top: 0.5em; border-top: 1px #ccc solid; text-align: center; }
` + msgCss + `
	</style>
</head>
<body>` + body + `
</body>
</html>`))
}

func (this *CaptchaValidator) validate(actionConfig *CaptchaAction, policyId int64, groupId int64, setId int64, originURL string, req requests.Request, writer http.ResponseWriter) (allow bool) {

	var captchaId = req.WAFRaw().FormValue("GOEDGE_WAF_CAPTCHA_ID")
	if len(captchaId) > 0 {
		var captchaCode = req.WAFRaw().FormValue("GOEDGE_WAF_CAPTCHA_CODE")
		if captcha.VerifyString(captchaId, captchaCode) {
			// 清除计数
			ttlcache.SharedCache.Delete(this.cacheKey(req))

			var life = CaptchaSeconds
			if actionConfig.Life > 0 {
				life = types.Int(actionConfig.Life)
			}

			// 加入到白名单
			SharedIPWhiteList.RecordIP("set:"+strconv.FormatInt(setId, 10), actionConfig.Scope, req.WAFServerId(), req.WAFRemoteIP(), time.Now().Unix()+int64(life), policyId, false, groupId, setId, "")

			http.Redirect(writer, req.WAFRaw(), originURL, http.StatusSeeOther)

			return false
		} else {
			// 增加计数
			if !this.IncreaseFails(req, actionConfig, policyId, groupId, setId) {
				return false
			}

			http.Redirect(writer, req.WAFRaw(), req.WAFRaw().URL.String(), http.StatusSeeOther)
		}
	}

	return true
}

// IncreaseFails 增加失败次数，以便后续操作
func (this *CaptchaValidator) IncreaseFails(req requests.Request, actionConfig *CaptchaAction, policyId int64, groupId int64, setId int64) (goNext bool) {
	var maxFails = actionConfig.MaxFails
	var failBlockTimeout = actionConfig.FailBlockTimeout
	if maxFails > 0 && failBlockTimeout > 0 {
		// 加上展示的计数
		maxFails *= 2

		var countFails = ttlcache.SharedCache.IncreaseInt64(this.cacheKey(req), 1, time.Now().Unix()+300, true)
		if int(countFails) >= maxFails {
			var useLocalFirewall = false

			if actionConfig.FailBlockScopeAll {
				useLocalFirewall = true
			}

			SharedIPBlackList.RecordIP(IPTypeAll, firewallconfigs.FirewallScopeService, req.WAFServerId(), req.WAFRemoteIP(), time.Now().Unix()+int64(failBlockTimeout), policyId, useLocalFirewall, groupId, setId, "CAPTCHA验证连续失败")
			return false
		}
	}
	return true
}

func (this *CaptchaValidator) cacheKey(req requests.Request) string {
	return "CAPTCHA:FAILS:" + req.WAFRemoteIP() + ":" + types.String(req.WAFServerId())
}
