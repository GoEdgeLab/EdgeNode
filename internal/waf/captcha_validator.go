package waf

import (
	"bytes"
	"encoding/base64"
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs/firewallconfigs"
	"github.com/TeaOSLab/EdgeNode/internal/ttlcache"
	"github.com/TeaOSLab/EdgeNode/internal/utils"
	"github.com/TeaOSLab/EdgeNode/internal/utils/fasttime"
	"github.com/TeaOSLab/EdgeNode/internal/waf/requests"
	wafutils "github.com/TeaOSLab/EdgeNode/internal/waf/utils"
	"github.com/dchest/captcha"
	"github.com/iwind/TeaGo/logs"
	"github.com/iwind/TeaGo/rands"
	"github.com/iwind/TeaGo/types"
	stringutil "github.com/iwind/TeaGo/utils/string"
	"net/http"
	"strings"
	"time"
)

var captchaValidator = NewCaptchaValidator()

type CaptchaValidator struct {
}

func NewCaptchaValidator() *CaptchaValidator {
	return &CaptchaValidator{}
}

func (this *CaptchaValidator) Run(req requests.Request, writer http.ResponseWriter, defaultCaptchaType firewallconfigs.ServerCaptchaType) {
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

	var captchaType = captchaActionConfig.CaptchaType
	if len(defaultCaptchaType) > 0 && defaultCaptchaType != firewallconfigs.ServerCaptchaTypeNone {
		captchaType = defaultCaptchaType
	}

	if req.WAFRaw().Method == http.MethodPost && len(req.WAFRaw().FormValue("GOEDGE_WAF_CAPTCHA_ID")) > 0 {
		switch captchaType {
		case firewallconfigs.CaptchaTypeOneClick:
			this.validateOneClickForm(captchaActionConfig, policyId, groupId, setId, originURL, req, writer)
		case firewallconfigs.CaptchaTypeSlide:
			this.validateSlideForm(captchaActionConfig, policyId, groupId, setId, originURL, req, writer)
		default:
			this.validateVerifyCodeForm(captchaActionConfig, policyId, groupId, setId, originURL, req, writer)
		}
	} else {
		// 增加计数
		CaptchaIncreaseFails(req, captchaActionConfig, policyId, groupId, setId, CaptchaPageCodeShow)
		this.show(captchaActionConfig, req, writer, captchaType)
	}
}

func (this *CaptchaValidator) show(actionConfig *CaptchaAction, req requests.Request, writer http.ResponseWriter, captchaType firewallconfigs.ServerCaptchaType) {
	switch captchaType {
	case firewallconfigs.CaptchaTypeOneClick:
		this.showOneClickForm(actionConfig, req, writer)
	case firewallconfigs.CaptchaTypeSlide:
		this.showSlideForm(actionConfig, req, writer)
	default:
		this.showVerifyCodesForm(actionConfig, req, writer)
	}
}

func (this *CaptchaValidator) showVerifyCodesForm(actionConfig *CaptchaAction, req requests.Request, writer http.ResponseWriter) {
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
		<p class="ui-prompt">` + msgPrompt + `</p>
		<input type="text" name="GOEDGE_WAF_CAPTCHA_CODE" id="GOEDGE_WAF_CAPTCHA_CODE" size="` + types.String(countLetters*17/10) + `" maxlength="` + types.String(countLetters) + `" autocomplete="off" z-index="1" class="input"/>
	</div>
	<div class="ui-button">
		<button type="submit" style="line-height:24px;margin-top:10px">` + msgButtonTitle + `</button>
	</div>
</form>
` + requestIdBox + `
` + msgFooter

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
	form { max-width: 20em; margin: 0 auto; text-align: center; }
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

func (this *CaptchaValidator) validateVerifyCodeForm(actionConfig *CaptchaAction, policyId int64, groupId int64, setId int64, originURL string, req requests.Request, writer http.ResponseWriter) (allow bool) {
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
			SharedIPWhiteList.RecordIP(wafutils.ComposeIPType(setId, req), actionConfig.Scope, req.WAFServerId(), req.WAFRemoteIP(), time.Now().Unix()+int64(life), policyId, false, groupId, setId, "")

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

func (this *CaptchaValidator) showOneClickForm(actionConfig *CaptchaAction, req requests.Request, writer http.ResponseWriter) {
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
	var msgRequestId string

	switch lang {
	case "zh-CN":
		msgTitle = "身份验证"
		msgPrompt = "我不是机器人"
		msgRequestId = "请求ID"
	case "zh-TW":
		msgTitle = "身份驗證"
		msgPrompt = "我不是機器人"
		msgRequestId = "請求ID"
	default:
		msgTitle = "Verify Yourself"
		msgPrompt = "I'm not a robot"
		msgRequestId = "Request ID"
	}

	var msgCss = ""
	var requestIdBox = `<address>` + msgRequestId + `: ` + req.Format("${requestId}") + `</address>`
	var msgFooter = ""

	// 默认设置
	if actionConfig.OneClickUIIsOn {
		if len(actionConfig.OneClickUIPrompt) > 0 {
			msgPrompt = actionConfig.OneClickUIPrompt
		}
		if len(actionConfig.OneClickUITitle) > 0 {
			msgTitle = actionConfig.OneClickUITitle
		}
		if len(actionConfig.OneClickUICss) > 0 {
			msgCss = actionConfig.OneClickUICss
		}
		if !actionConfig.OneClickUIShowRequestId {
			requestIdBox = ""
		}
		if len(actionConfig.OneClickUIFooter) > 0 {
			msgFooter = actionConfig.OneClickUIFooter
		}
	}

	var captchaId = stringutil.Md5(req.WAFRemoteIP() + "@" + stringutil.Rand(32))
	var nonce = rands.Int64()
	if !ttlcache.SharedInt64Cache.Write("WAF_CAPTCHA:"+captchaId, nonce, fasttime.Now().Unix()+600) {
		return
	}

	var body = `<form method="POST" id="ui-form">
	<input type="hidden" name="GOEDGE_WAF_CAPTCHA_ID" value="` + captchaId + `"/>
	<div class="ui-input">
		<div class="ui-checkbox" id="checkbox"></div>
		<p class="ui-prompt">` + msgPrompt + `</p>
	</div>
</form>
` + requestIdBox + `
` + msgFooter

	// Body
	if actionConfig.OneClickUIIsOn {
		if len(actionConfig.OneClickUIBody) > 0 {
			var index = strings.Index(actionConfig.OneClickUIBody, "${body}")
			if index < 0 {
				body = actionConfig.OneClickUIBody + body
			} else {
				body = actionConfig.OneClickUIBody[:index] + body + actionConfig.OneClickUIBody[index+7:] // 7是"${body}"的长度
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
	window.addEventListener("load",function(){var t=document.getElementById("checkbox"),n=!1;t.addEventListener("click",function(){var e;t.className="ui-checkbox checked",n||(n=!0,(e=document.createElement("input")).setAttribute("name","nonce"),e.setAttribute("type","hidden"),e.setAttribute("value","` + types.String(nonce) + `"),document.getElementById("ui-form").appendChild(e),document.getElementById("ui-form").submit())})});
	</script>
	<style type="text/css">
	form { max-width: 20em; margin: 0 auto; text-align: center; }
    .ui-input { position: relative; padding-top: 1em; height: 2.2em; background: #eee; }
    .ui-checkbox { width: 16px; height: 16px; border: 1px #999 solid; float: left; margin-left: 1em; cursor: pointer; }
    .ui-checkbox.checked { background: #276AC6; }
    .ui-prompt { float: left; margin: 0; margin-left: 0.5em; padding: 0; line-height: 1.2; }
	address { margin-top: 1em; padding-top: 0.5em; border-top: 1px #ccc solid; text-align: center; clear: both; }
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

func (this *CaptchaValidator) validateOneClickForm(actionConfig *CaptchaAction, policyId int64, groupId int64, setId int64, originURL string, req requests.Request, writer http.ResponseWriter) (allow bool) {
	var captchaId = req.WAFRaw().FormValue("GOEDGE_WAF_CAPTCHA_ID")
	var nonce = req.WAFRaw().FormValue("nonce")
	if len(captchaId) > 0 {
		var key = "WAF_CAPTCHA:" + captchaId
		var cacheItem = ttlcache.SharedInt64Cache.Read(key)
		ttlcache.SharedInt64Cache.Delete(key)
		if cacheItem != nil {
			// 清除计数
			CaptchaDeleteCacheKey(req)

			if cacheItem.Value == types.Int64(nonce) {
				var life = CaptchaSeconds
				if actionConfig.Life > 0 {
					life = types.Int(actionConfig.Life)
				}

				// 加入到白名单
				SharedIPWhiteList.RecordIP(wafutils.ComposeIPType(setId, req), actionConfig.Scope, req.WAFServerId(), req.WAFRemoteIP(), time.Now().Unix()+int64(life), policyId, false, groupId, setId, "")

				req.ProcessResponseHeaders(writer.Header(), http.StatusSeeOther)
				http.Redirect(writer, req.WAFRaw(), originURL, http.StatusSeeOther)

				return false
			}
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

func (this *CaptchaValidator) showSlideForm(actionConfig *CaptchaAction, req requests.Request, writer http.ResponseWriter) {
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
	var msgRequestId string

	switch lang {
	case "zh-CN":
		msgTitle = "身份验证"
		msgPrompt = "滑动上面方块到右侧解锁"
		msgRequestId = "请求ID"
	case "zh-TW":
		msgTitle = "身份驗證"
		msgPrompt = "滑動上面方塊到右側解鎖"
		msgRequestId = "請求ID"
	default:
		msgTitle = "Verify Yourself"
		msgPrompt = "Slide to Unlock"
		msgRequestId = "Request ID"
	}

	var msgCss = ""
	var requestIdBox = `<address>` + msgRequestId + `: ` + req.Format("${requestId}") + `</address>`
	var msgFooter = ""

	// 默认设置
	if actionConfig.OneClickUIIsOn {
		if len(actionConfig.OneClickUIPrompt) > 0 {
			msgPrompt = actionConfig.OneClickUIPrompt
		}
		if len(actionConfig.OneClickUITitle) > 0 {
			msgTitle = actionConfig.OneClickUITitle
		}
		if len(actionConfig.OneClickUICss) > 0 {
			msgCss = actionConfig.OneClickUICss
		}
		if !actionConfig.OneClickUIShowRequestId {
			requestIdBox = ""
		}
		if len(actionConfig.OneClickUIFooter) > 0 {
			msgFooter = actionConfig.OneClickUIFooter
		}
	}

	var captchaId = stringutil.Md5(req.WAFRemoteIP() + "@" + stringutil.Rand(32))
	var nonce = rands.Int64()
	if !ttlcache.SharedInt64Cache.Write("WAF_CAPTCHA:"+captchaId, nonce, fasttime.Now().Unix()+600) {
		return
	}

	var body = `<form method="POST" id="ui-form">
	<input type="hidden" name="GOEDGE_WAF_CAPTCHA_ID" value="` + captchaId + `"/>
	 <div class="ui-input" id="input">
      	<div class="ui-progress-bar" id="progress-bar"></div>
        <div class="ui-handler" id="handler"></div>
		<div class="ui-handler-placeholder" id="handler-placeholder"></div>
 		<img alt="" src="data:image/jpeg;charset=utf-8;base64,iVBORw0KGgoAAAANSUhEUgAAAIAAAACACAYAAADDPmHLAAAACXBIWXMAAAsTAAALEwEAmpwYAAACHklEQVR4nO3cu4oUQRgF4EI3UhDXRNRQUFnBC97XwIcRfQf3ZfYdTAxMFMwMzUw18q6JCOpaxayBTItdDd3V9v99cOKt5pyZnWlmJiUAAAAAAAAAAAAAAAAAAAAAgOgO5pzLOdn6IEzvXs6bnL39PM+53PRETKaUv9eRLzm3G56LCWykPx/5XSPYbnY6Rncm/b383/mcc6vVARnXqfTvARjBwpUXfH1HcLPRGRlRebVf/tf3GcGnZASLVF7olUe4Z4LAakZQnglutDkmYzICjAAjINWP4HqbYzImI8AIqBvBx2QEi1Q7gmttjsmYjAAjoH4EV9sckzEZAUaAEZDqRvAh50qbYzKm5iMoH3F+kPNi/w/I9PmW+g2g5G3Ohc4mBziQ87jij8s8Ur6TcLqjz2r3Z3AxMixP1uus93AGFyLD8jPn6HqldR7N4EJk+AA21yutszODC5FhedbRZ7UjOS9ncDFSl/LO4WxHn4Mcy9nN+TqDC5N+5Y9yQ6j80sWmTJ47qe5GkFvCC3IxrW7s9CnfR8YWRvmBKT8w5Qem/MAuJeWHVcp/l5QfkvIDU35gyg+spnzfDF4Y5QdWfjtQ+UEpPzDlB1bKf5+UH5LyAzufVu/f+77P90mehSmfylV+UIdzfiRP+2GdSB754b1Oyg/tblJ+eGUEr9Kq+O85T3O2mp6IJo7nHGp9CAAAAAAAAAAAAAAAAAAAAADgf/ILsUB70laSdmQAAAAASUVORK5CYII="/>
    </div>
	<p class="ui-prompt">` + msgPrompt + `</p>
</form>
` + requestIdBox + `
` + msgFooter

	// Body
	if actionConfig.OneClickUIIsOn {
		if len(actionConfig.OneClickUIBody) > 0 {
			var index = strings.Index(actionConfig.OneClickUIBody, "${body}")
			if index < 0 {
				body = actionConfig.OneClickUIBody + body
			} else {
				body = actionConfig.OneClickUIBody[:index] + body + actionConfig.OneClickUIBody[index+7:] // 7是"${body}"的长度
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
window.addEventListener("load",function(){var n=document.getElementById("input"),s=document.getElementById("handler"),o=document.getElementById("progress-bar"),d=!1,u=0,t=n.offsetLeft,c=s.offsetLeft,i=n.offsetWidth-s.offsetWidth-s.offsetLeft,f=!1;function e(e){e.preventDefault(),d=!0,u=null!=e.touches&&0<e.touches.length?e.touches[0].clientX-t:e.offsetX}function l(e){var t;d&&(t=e.x,null!=e.touches&&0<e.touches.length&&(t=e.touches[0].clientX),(t=t-n.offsetLeft-u)<c?t=c:i<t&&(t=i),s.style.cssText="margin-left: "+t+"px",0<t&&(o.style.cssText="width: "+(t+s.offsetWidth+4)+"px"))}function r(e){var t;d=d&&!1,s.offsetLeft<i-4?(s.style.cssText="margin-left: "+c+"px",n.style.cssText="background: #eee",o.style.cssText="width: 0px"):(s.style.cssText="margin-left: "+i+"px",n.style.cssText="background: #a5dc86",f||(f=!0,(t=document.createElement("input")).setAttribute("name","nonce"),t.setAttribute("type","hidden"),t.setAttribute("value","` + types.String(nonce) + `"),document.getElementById("ui-form").appendChild(t),document.getElementById("ui-form").submit()))}void 0!==document.ontouchstart?(s.addEventListener("touchstart",e),document.addEventListener("touchmove",l),document.addEventListener("touchend",r)):(s.addEventListener("mousedown",e),window.addEventListener("mousemove",l),window.addEventListener("mouseup",r))});
	</script>
	<style type="text/css">
	form { max-width: 20em; margin: 5em auto; text-align: center; }
 		.ui-input {
            height: 4em;
            background: #eee;
			border: 1px #ccc solid;
            text-align: left;
            position: relative;
        }

        .ui-input .ui-progress-bar {
            background: #a5dc86;
            position: absolute;
            top: 0;
            left: 0;
            bottom: 0;
        }

        .ui-handler, .ui-handler-placeholder {
            width: 3.6em;
            height: 3.6em;
            margin: 0.2em;
            background: #276AC6;
            border-radius: 0.6em;
            display: inline-block;
            cursor: pointer;
            z-index: 10;
            position: relative;
        }

        .ui-handler-placeholder {
            background: none;
            border: 1px #ccc dashed;
            position: absolute;
            right: -1px;
            top: 0;
            bottom: 0;
        }

        .ui-input img {
            position: absolute;
            top: 0;
            bottom: 0;
            height: 100%;
            left: 8em;
            opacity: 5%;
        }
    .ui-prompt { float: left; margin: 1em 0; padding: 0; line-height: 1.2; }
	address { margin-top: 1em; padding-top: 0.5em; border-top: 1px #ccc solid; text-align: center; clear: both; }
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

func (this *CaptchaValidator) validateSlideForm(actionConfig *CaptchaAction, policyId int64, groupId int64, setId int64, originURL string, req requests.Request, writer http.ResponseWriter) (allow bool) {
	var captchaId = req.WAFRaw().FormValue("GOEDGE_WAF_CAPTCHA_ID")
	var nonce = req.WAFRaw().FormValue("nonce")
	if len(captchaId) > 0 {
		var key = "WAF_CAPTCHA:" + captchaId
		var cacheItem = ttlcache.SharedInt64Cache.Read(key)
		ttlcache.SharedInt64Cache.Delete(key)
		if cacheItem != nil {
			// 清除计数
			CaptchaDeleteCacheKey(req)

			if cacheItem.Value == types.Int64(nonce) {
				var life = CaptchaSeconds
				if actionConfig.Life > 0 {
					life = types.Int(actionConfig.Life)
				}

				// 加入到白名单
				SharedIPWhiteList.RecordIP(wafutils.ComposeIPType(setId, req), actionConfig.Scope, req.WAFServerId(), req.WAFRemoteIP(), time.Now().Unix()+int64(life), policyId, false, groupId, setId, "")

				req.ProcessResponseHeaders(writer.Header(), http.StatusSeeOther)
				http.Redirect(writer, req.WAFRaw(), originURL, http.StatusSeeOther)

				return false
			}
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
