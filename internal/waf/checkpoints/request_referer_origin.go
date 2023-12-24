package checkpoints

import (
	"github.com/TeaOSLab/EdgeNode/internal/waf/requests"
	"github.com/TeaOSLab/EdgeNode/internal/waf/utils"
	"github.com/iwind/TeaGo/maps"
)

type RequestRefererOriginCheckpoint struct {
	Checkpoint
}

func (this *RequestRefererOriginCheckpoint) RequestValue(req requests.Request, param string, options maps.Map, ruleId int64) (value any, hasRequestBody bool, sysErr error, userErr error) {
	var s []string

	var referer = req.WAFRaw().Referer()
	if len(referer) > 0 {
		s = append(s, referer)
	}

	var origin = req.WAFRaw().Header.Get("Origin")
	if len(origin) > 0 {
		s = append(s, origin)
	}

	if len(s) > 0 {
		value = s
	} else {
		value = ""
	}

	return
}

func (this *RequestRefererOriginCheckpoint) ResponseValue(req requests.Request, resp *requests.Response, param string, options maps.Map, ruleId int64) (value any, hasRequestBody bool, sysErr error, userErr error) {
	if this.IsRequest() {
		return this.RequestValue(req, param, options, ruleId)
	}
	return
}

func (this *RequestRefererOriginCheckpoint) CacheLife() utils.CacheLife {
	return utils.CacheMiddleLife
}
