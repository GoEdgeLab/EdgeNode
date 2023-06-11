package waf

import (
	teaconst "github.com/TeaOSLab/EdgeNode/internal/const"
	"github.com/TeaOSLab/EdgeNode/internal/utils"
	"github.com/TeaOSLab/EdgeNode/internal/waf/requests"
	"github.com/iwind/TeaGo/Tea"
	"github.com/iwind/TeaGo/logs"
	"github.com/iwind/TeaGo/rands"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"time"
)

// url client configure
var urlPrefixReg = regexp.MustCompile("^(?i)(http|https)://")
var httpClient = utils.SharedHttpClient(5 * time.Second)

type BlockAction struct {
	BaseAction

	StatusCode int    `yaml:"statusCode" json:"statusCode"`
	Body       string `yaml:"body" json:"body"` // supports HTML
	URL        string `yaml:"url" json:"url"`
	Timeout    int32  `yaml:"timeout" json:"timeout"`
	TimeoutMax int32  `yaml:"timeoutMax" json:"timeoutMax"`
	Scope      string `yaml:"scope" json:"scope"`
}

func (this *BlockAction) Init(waf *WAF) error {
	if waf.DefaultBlockAction != nil {
		if this.StatusCode <= 0 {
			this.StatusCode = waf.DefaultBlockAction.StatusCode
		}
		if len(this.Body) == 0 {
			this.Body = waf.DefaultBlockAction.Body
		}
		if len(this.URL) == 0 {
			this.URL = waf.DefaultBlockAction.URL
		}
		if this.Timeout <= 0 {
			this.Timeout = waf.DefaultBlockAction.Timeout
			this.TimeoutMax = waf.DefaultBlockAction.TimeoutMax // 只有没有填写封锁时长的时候才会使用默认的封锁时长最大值
		}
	}
	return nil
}

func (this *BlockAction) Code() string {
	return ActionBlock
}

func (this *BlockAction) IsAttack() bool {
	return true
}

func (this *BlockAction) WillChange() bool {
	return true
}

func (this *BlockAction) Perform(waf *WAF, group *RuleGroup, set *RuleSet, request requests.Request, writer http.ResponseWriter) (continueRequest bool, goNextSet bool) {
	// 加入到黑名单
	var timeout = this.Timeout
	if timeout <= 0 {
		timeout = 300 // 默认封锁300秒
	}

	// 随机时长
	var timeoutMax = this.TimeoutMax
	if timeoutMax > timeout {
		timeout = timeout + int32(rands.Int64()%int64(timeoutMax-timeout+1))
	}

	SharedIPBlackList.RecordIP(IPTypeAll, this.Scope, request.WAFServerId(), request.WAFRemoteIP(), time.Now().Unix()+int64(timeout), waf.Id, waf.UseLocalFirewall, group.Id, set.Id, "")

	if writer != nil {
		// close the connection
		defer request.WAFClose()

		// output response
		if this.StatusCode > 0 {
			request.ProcessResponseHeaders(writer.Header(), this.StatusCode)
			writer.WriteHeader(this.StatusCode)
		} else {
			request.ProcessResponseHeaders(writer.Header(), http.StatusForbidden)
			writer.WriteHeader(http.StatusForbidden)
		}
		if len(this.URL) > 0 {
			if urlPrefixReg.MatchString(this.URL) {
				req, err := http.NewRequest(http.MethodGet, this.URL, nil)
				if err != nil {
					logs.Error(err)
					return false, false
				}
				req.Header.Set("User-Agent", teaconst.GlobalProductName+"/"+teaconst.Version)

				resp, err := httpClient.Do(req)
				if err != nil {
					logs.Error(err)
					return false, false
				}
				defer func() {
					_ = resp.Body.Close()
				}()

				for k, v := range resp.Header {
					for _, v1 := range v {
						writer.Header().Add(k, v1)
					}
				}

				buf := utils.BytePool1k.Get()
				_, _ = io.CopyBuffer(writer, resp.Body, buf)
				utils.BytePool1k.Put(buf)
			} else {
				path := this.URL
				if !filepath.IsAbs(this.URL) {
					path = Tea.Root + string(os.PathSeparator) + path
				}

				data, err := os.ReadFile(path)
				if err != nil {
					logs.Error(err)
					return false, false
				}
				_, _ = writer.Write(data)
			}
			return false, false
		}
		if len(this.Body) > 0 {
			_, _ = writer.Write([]byte(this.Body))
		} else {
			_, _ = writer.Write([]byte("The request is blocked by " + teaconst.ProductName))
		}
	}

	return false, false
}
