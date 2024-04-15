package waf

import (
	"github.com/TeaOSLab/EdgeNode/internal/remotelogs"
	"github.com/TeaOSLab/EdgeNode/internal/utils"
	"github.com/TeaOSLab/EdgeNode/internal/waf/requests"
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

func (this *Post307Action) Perform(waf *WAF, group *RuleGroup, set *RuleSet, request requests.Request, writer http.ResponseWriter) PerformResult {
	const cookieName = "WAF_VALIDATOR_ID"

	// 仅限于POST
	if request.WAFRaw().Method != http.MethodPost {
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

	// 判断是否有Cookie
	cookie, cookieErr := request.WAFRaw().Cookie(cookieName)
	if cookieErr == nil && cookie != nil {
		var remoteIP string
		var life int64
		var setId int64
		var policyId int64
		var groupId int64
		var timestamp int64

		var infoArg = &InfoArg{}
		var success bool
		decodeErr := infoArg.Decode(cookie.Value)
		if decodeErr == nil && infoArg.IsValid() {
			success = true

			remoteIP = infoArg.RemoteIP
			life = int64(infoArg.Life)
			setId = infoArg.SetId
			policyId = infoArg.PolicyId
			groupId = infoArg.GroupId
			timestamp = infoArg.Timestamp
		} else {
			// 兼容老版本
			m, decodeMapErr := utils.SimpleDecryptMap(cookie.Value)
			if decodeMapErr == nil {
				success = true

				remoteIP = m.GetString("remoteIP")
				timestamp = m.GetInt64("timestamp")
				life = m.GetInt64("life")
				setId = m.GetInt64("setId")
				groupId = m.GetInt64("groupId")
				policyId = m.GetInt64("policyId")
			}
		}

		if success && remoteIP == request.WAFRemoteIP() && time.Now().Unix() < timestamp+10 {
			if life <= 0 {
				life = 600 // 默认10分钟
			}
			SharedIPWhiteList.RecordIP("set:"+types.String(setId), this.Scope, request.WAFServerId(), request.WAFRemoteIP(), time.Now().Unix()+life, policyId, false, groupId, setId, "")
			return PerformResult{
				ContinueRequest: true,
			}
		}
	}

	var m = &InfoArg{
		Timestamp:        time.Now().Unix(),
		Life:             this.Life,
		Scope:            this.Scope,
		PolicyId:         waf.Id,
		GroupId:          group.Id,
		SetId:            set.Id,
		RemoteIP:         request.WAFRemoteIP(),
		UseLocalFirewall: false,
	}
	info, err := utils.SimpleEncryptObject(m)
	if err != nil {
		remotelogs.Error("WAF_POST_307_ACTION", "encode info failed: "+err.Error())
		return PerformResult{
			ContinueRequest: true,
		}
	}

	// 清空请求内容
	var req = request.WAFRaw()
	if req.ContentLength > 0 && req.Body != nil {
		var buf = utils.BytePool16k.Get()
		_, _ = io.CopyBuffer(io.Discard, req.Body, buf.Bytes)
		utils.BytePool16k.Put(buf)
		_ = req.Body.Close()
	}

	// 设置Cookie
	http.SetCookie(writer, &http.Cookie{
		Name:   cookieName,
		Path:   "/",
		MaxAge: 10,
		Value:  info,
	})

	request.DisableStat()
	request.ProcessResponseHeaders(writer.Header(), http.StatusTemporaryRedirect)
	http.Redirect(writer, request.WAFRaw(), request.WAFRaw().URL.String(), http.StatusTemporaryRedirect)

	flusher, ok := writer.(http.Flusher)
	if ok {
		flusher.Flush()
	}

	return PerformResult{}
}
