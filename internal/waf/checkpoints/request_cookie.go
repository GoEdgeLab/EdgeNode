package checkpoints

import (
	"github.com/TeaOSLab/EdgeNode/internal/waf/requests"
	"github.com/TeaOSLab/EdgeNode/internal/waf/utils"
	"github.com/iwind/TeaGo/maps"
)

type RequestCookieCheckpoint struct {
	Checkpoint
}

func (this *RequestCookieCheckpoint) RequestValue(req requests.Request, param string, options maps.Map, ruleId int64) (value any, hasRequestBody bool, sysErr error, userErr error) {
	cookie, err := req.WAFRaw().Cookie(param)
	if err != nil {
		value = ""
		return
	}

	value = cookie.Value
	return
}

func (this *RequestCookieCheckpoint) ResponseValue(req requests.Request, resp *requests.Response, param string, options maps.Map, ruleId int64) (value any, hasRequestBody bool, sysErr error, userErr error) {
	if this.IsRequest() {
		return this.RequestValue(req, param, options, ruleId)
	}
	return
}

func (this *RequestCookieCheckpoint) CacheLife() utils.CacheLife {
	return utils.CacheMiddleLife
}
