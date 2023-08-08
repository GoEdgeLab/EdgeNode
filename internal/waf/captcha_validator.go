package waf

import (
	"bytes"
	"encoding/base64"
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
		req.ProcessResponseHeaders(writer.Header(), http.StatusBadRequest)
		writer.WriteHeader(http.StatusBadRequest)
		_, _ = writer.Write([]byte("invalid request"))
		return
	}
	m, err := utils.SimpleDecryptMap(info)
	if err != nil {
		req.ProcessResponseHeaders(writer.Header(), http.StatusBadRequest)
		writer.WriteHeader(http.StatusBadRequest)
		_, _ = writer.Write([]byte("invalid request"))
		return
	}

	var timestamp = m.GetInt64("timestamp")
	if timestamp < time.Now().Unix()-600 { // 10分钟之后信息过期
		req.ProcessResponseHeaders(writer.Header(), http.StatusTemporaryRedirect)
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
		req.ProcessResponseHeaders(writer.Header(), http.StatusTemporaryRedirect)
		http.Redirect(writer, req.WAFRaw(), originURL, http.StatusTemporaryRedirect)
		return
	}
	var actionConfig = waf.FindAction(actionId)
	if actionConfig == nil {
		req.ProcessResponseHeaders(writer.Header(), http.StatusTemporaryRedirect)
		http.Redirect(writer, req.WAFRaw(), originURL, http.StatusTemporaryRedirect)
		return
	}
	captchaActionConfig, ok := actionConfig.(*CaptchaAction)
	if !ok {
		req.ProcessResponseHeaders(writer.Header(), http.StatusTemporaryRedirect)
		http.Redirect(writer, req.WAFRaw(), originURL, http.StatusTemporaryRedirect)
		return
	}

	if req.WAFRaw().Method == http.MethodPost && len(req.WAFRaw().FormValue("GOEDGE_WAF_CAPTCHA_ID")) > 0 {
		this.validate(captchaActionConfig, policyId, groupId, setId, originURL, req, writer)
	} else {
		// 增加计数
		CaptchaIncreaseFails(req, captchaActionConfig, policyId, groupId, setId, CaptchaPageCodeShow)
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

	var msgTitle string
	var msgPrompt string
	var msgButtonTitle string
	var msgRequestId string

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
	case "zh-TW":
		msgTitle = "身份驗證"
		msgPrompt = "請輸入上面的驗證碼"
		msgButtonTitle = "提交驗證"
		msgRequestId = "請求ID"
	default:
		msgTitle = "Verify Yourself"
		msgPrompt = "Input verify code above:"
		msgButtonTitle = "Verify Yourself"
		msgRequestId = "Request ID"
	}

	var msgCss = ""
	var requestIdBox = `<address>` + msgRequestId + `: ` + req.Format("${requestId}") + `</address>`
	var msgFooter = ""

	// 默认设置
	if actionConfig.UIIsOn {
		if len(actionConfig.UIPrompt) > 0 {
			msgPrompt = actionConfig.UIPrompt
		}
		if len(actionConfig.UIButtonTitle) > 0 {
			msgButtonTitle = actionConfig.UIButtonTitle
		}
		if len(actionConfig.UITitle) > 0 {
			msgTitle = actionConfig.UITitle
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
	}

	var body = `<form method="POST">
	<input type="hidden" name="GOEDGE_WAF_CAPTCHA_ID" value="` + captchaId + `"/>
	<div class="ui-image">
		<img src="data:image/png;base64, ` + base64.StdEncoding.EncodeToString(buf.Bytes()) + `"/>` + `
	</div>
	<div class="ui-input">
		<p>` + msgPrompt + `</p>
		<input type="text" name="GOEDGE_WAF_CAPTCHA_CODE" id="GOEDGE_WAF_CAPTCHA_CODE" size="` + types.String(countLetters*17/10) + `" maxlength="` + types.String(countLetters) + `" autocomplete="off" z-index="1" class="input"/>
	</div>
	<div class="ui-button">
		<button type="submit" style="line-height:24px;margin-top:10px">` + msgButtonTitle + `</button>
	</div>
</form>
` + requestIdBox + `
` + msgFooter + ``

	// Body
	if actionConfig.UIIsOn {
		if len(actionConfig.UIBody) > 0 {
			var index = strings.Index(actionConfig.UIBody, "${body}")
			if index < 0 {
				body = actionConfig.UIBody + body
			} else {
				body = actionConfig.UIBody[:index] + body + actionConfig.UIBody[index+7:] // 7是"${body}"的长度
			}
		}
	}

	var msgHTML = `<!DOCTYPE html>
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
	.input { font-size:16px;line-height:24px; letter-spacing:0.2em; min-width: 5em; text-align: center; }
	address { margin-top: 1em; padding-top: 0.5em; border-top: 1px #ccc solid; text-align: center; }
` + msgCss + `
	</style>
</head>
<body>` + body + `
</body>
</html>`

	req.ProcessResponseHeaders(writer.Header(), http.StatusOK)
	writer.Header().Set("Content-Type", "text/html; charset=utf-8")
	writer.Header().Set("Content-Length", types.String(len(msgHTML)))
	writer.WriteHeader(http.StatusOK)
	_, _ = writer.Write([]byte(msgHTML))
}

func (this *CaptchaValidator) validate(actionConfig *CaptchaAction, policyId int64, groupId int64, setId int64, originURL string, req requests.Request, writer http.ResponseWriter) (allow bool) {

	var captchaId = req.WAFRaw().FormValue("GOEDGE_WAF_CAPTCHA_ID")
	if len(captchaId) > 0 {
		var captchaCode = req.WAFRaw().FormValue("GOEDGE_WAF_CAPTCHA_CODE")
		if captcha.VerifyString(captchaId, captchaCode) {
			// 清除计数
			CaptchaDeleteCacheKey(req)

			var life = CaptchaSeconds
			if actionConfig.Life > 0 {
				life = types.Int(actionConfig.Life)
			}

			// 加入到白名单
			SharedIPWhiteList.RecordIP("set:"+strconv.FormatInt(setId, 10), actionConfig.Scope, req.WAFServerId(), req.WAFRemoteIP(), time.Now().Unix()+int64(life), policyId, false, groupId, setId, "")

			req.ProcessResponseHeaders(writer.Header(), http.StatusSeeOther)
			http.Redirect(writer, req.WAFRaw(), originURL, http.StatusSeeOther)

			return false
		} else {
			// 增加计数
			if !CaptchaIncreaseFails(req, actionConfig, policyId, groupId, setId, CaptchaPageCodeSubmit) {
				return false
			}

			req.ProcessResponseHeaders(writer.Header(), http.StatusSeeOther)
			http.Redirect(writer, req.WAFRaw(), req.WAFRaw().URL.String(), http.StatusSeeOther)
		}
	}

	return true
}
