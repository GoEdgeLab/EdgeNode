package waf

import (
	"github.com/TeaOSLab/EdgeNode/internal/remotelogs"
	"github.com/TeaOSLab/EdgeNode/internal/utils"
	"github.com/TeaOSLab/EdgeNode/internal/waf/requests"
	"github.com/iwind/TeaGo/maps"
	"github.com/iwind/TeaGo/types"
	"io"
	"net/http"
	"time"
)

type Post307Action struct {
	Life  int32  `yaml:"life" json:"life"`
	Scope string `yaml:"scope" json:"scope"`

	BaseAction
}

func (this *Post307Action) Init(waf *WAF) error {
	return nil
}

func (this *Post307Action) Code() string {
	return ActionPost307
}

func (this *Post307Action) IsAttack() bool {
	return false
}

func (this *Post307Action) WillChange() bool {
	return true
}

func (this *Post307Action) Perform(waf *WAF, group *RuleGroup, set *RuleSet, request requests.Request, writer http.ResponseWriter) (continueRequest bool, goNextSet bool) {
	var cookieName = "WAF_VALIDATOR_ID"

	// 仅限于POST
	if request.WAFRaw().Method != http.MethodPost {
		return true, false
	}

	// 是否已经在白名单中
	if SharedIPWhiteList.Contains("set:"+types.String(set.Id), this.Scope, request.WAFServerId(), request.WAFRemoteIP()) {
		return true, false
	}

	// 判断是否有Cookie
	cookie, err := request.WAFRaw().Cookie(cookieName)
	if err == nil && cookie != nil {
		m, err := utils.SimpleDecryptMap(cookie.Value)
		if err == nil && m.GetString("remoteIP") == request.WAFRemoteIP() && time.Now().Unix() < m.GetInt64("timestamp")+10 {
			var life = m.GetInt64("life")
			if life <= 0 {
				life = 600 // 默认10分钟
			}
			var setId = types.String(m.GetInt64("setId"))
			SharedIPWhiteList.RecordIP("set:"+setId, this.Scope, request.WAFServerId(), request.WAFRemoteIP(), time.Now().Unix()+life, m.GetInt64("policyId"), false, m.GetInt64("groupId"), m.GetInt64("setId"), "")
			return true, false
		}
	}

	var m = maps.Map{
		"timestamp": time.Now().Unix(),
		"life":      this.Life,
		"scope":     this.Scope,
		"policyId":  waf.Id,
		"groupId":   group.Id,
		"setId":     set.Id,
		"remoteIP":  request.WAFRemoteIP(),
	}
	info, err := utils.SimpleEncryptMap(m)
	if err != nil {
		remotelogs.Error("WAF_POST_307_ACTION", "encode info failed: "+err.Error())
		return true, false
	}

	// 清空请求内容
	var req = request.WAFRaw()
	if req.ContentLength > 0 && req.Body != nil {
		_, _ = io.Copy(io.Discard, req.Body)
		_ = req.Body.Close()
	}

	// 设置Cookie
	http.SetCookie(writer, &http.Cookie{
		Name:   cookieName,
		Path:   "/",
		MaxAge: 10,
		Value:  info,
	})

	request.ProcessResponseHeaders(writer.Header(), http.StatusTemporaryRedirect)
	http.Redirect(writer, request.WAFRaw(), request.WAFRaw().URL.String(), http.StatusTemporaryRedirect)

	flusher, ok := writer.(http.Flusher)
	if ok {
		flusher.Flush()
	}

	return false, false
}
