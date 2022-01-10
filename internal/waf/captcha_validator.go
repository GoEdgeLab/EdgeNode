package waf

import (
	"bytes"
	"encoding/base64"
	"github.com/TeaOSLab/EdgeNode/internal/utils"
	"github.com/TeaOSLab/EdgeNode/internal/utils/jsonutils"
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

func (this *CaptchaValidator) Run(request requests.Request, writer http.ResponseWriter) {
	var info = request.WAFRaw().URL.Query().Get("info")
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

	timestamp := m.GetInt64("timestamp")
	if timestamp < time.Now().Unix()-600 { // 10分钟之后信息过期
		http.Redirect(writer, request.WAFRaw(), m.GetString("url"), http.StatusTemporaryRedirect)
		return
	}

	var actionConfig = &CaptchaAction{}
	err = jsonutils.MapToObject(m.GetMap("action"), actionConfig)
	if err != nil {
		http.Redirect(writer, request.WAFRaw(), m.GetString("url"), http.StatusTemporaryRedirect)
		return
	}

	var setId = m.GetInt64("setId")
	var originURL = m.GetString("url")

	if request.WAFRaw().Method == http.MethodPost && len(request.WAFRaw().FormValue("GOEDGE_WAF_CAPTCHA_ID")) > 0 {
		this.validate(actionConfig, m.GetInt64("policyId"), m.GetInt64("groupId"), setId, originURL, request, writer)
	} else {
		this.show(actionConfig, request, writer)
	}
}

func (this *CaptchaValidator) show(actionConfig *CaptchaAction, request requests.Request, writer http.ResponseWriter) {
	// show captcha
	captchaId := captcha.NewLen(6)
	buf := bytes.NewBuffer([]byte{})
	err := captcha.WriteImage(buf, captchaId, 200, 100)
	if err != nil {
		logs.Error(err)
		return
	}

	var lang = actionConfig.Language
	if len(lang) == 0 {
		acceptLanguage := request.WAFRaw().Header.Get("Accept-Language")
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

	writer.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = writer.Write([]byte(`<!DOCTYPE html>
<html>
<head>
	<title>` + msgTitle + `</title>
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
	</style>
</head>
<body>
<form method="POST">
	<input type="hidden" name="GOEDGE_WAF_CAPTCHA_ID" value="` + captchaId + `"/>
	<img src="data:image/png;base64, ` + base64.StdEncoding.EncodeToString(buf.Bytes()) + `"/>` + `
	<div>
		<p>` + msgPrompt + `</p>
		<input type="text" name="GOEDGE_WAF_CAPTCHA_CODE" id="GOEDGE_WAF_CAPTCHA_CODE" maxlength="6" autocomplete="off" z-index="1" class="input"/>
	</div>
	<div>
		<button type="submit" style="line-height:24px;margin-top:10px">` + msgButtonTitle + `</button>
	</div>
</form>
<address>` + msgRequestId + `: ` + request.Format("${requestId}") + `</address>
</body>
</html>`))
}

func (this *CaptchaValidator) validate(actionConfig *CaptchaAction, policyId int64, groupId int64, setId int64, originURL string, request requests.Request, writer http.ResponseWriter) (allow bool) {
	captchaId := request.WAFRaw().FormValue("GOEDGE_WAF_CAPTCHA_ID")
	if len(captchaId) > 0 {
		captchaCode := request.WAFRaw().FormValue("GOEDGE_WAF_CAPTCHA_CODE")
		if captcha.VerifyString(captchaId, captchaCode) {
			var life = CaptchaSeconds
			if actionConfig.Life > 0 {
				life = types.Int(actionConfig.Life)
			}

			// 加入到白名单
			SharedIPWhiteList.RecordIP("set:"+strconv.FormatInt(setId, 10), actionConfig.Scope, request.WAFServerId(), request.WAFRemoteIP(), time.Now().Unix()+int64(life), policyId, false, groupId, setId, "")

			http.Redirect(writer, request.WAFRaw(), originURL, http.StatusSeeOther)

			return false
		} else {
			http.Redirect(writer, request.WAFRaw(), request.WAFRaw().URL.String(), http.StatusSeeOther)
		}
	}

	return true
}
