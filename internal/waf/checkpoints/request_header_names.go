package checkpoints

import (
	"github.com/TeaOSLab/EdgeNode/internal/waf/requests"
	"github.com/TeaOSLab/EdgeNode/internal/waf/utils"
	"github.com/iwind/TeaGo/maps"
	"strings"
)

type RequestHeaderNamesCheckpoint struct {
	Checkpoint
}

func (this *RequestHeaderNamesCheckpoint) RequestValue(req requests.Request, param string, options maps.Map, ruleId int64) (value any, hasRequestBody bool, sysErr error, userErr error) {
	var headerNames = []string{}
	for k := range req.WAFRaw().Header {
		headerNames = append(headerNames, k)
	}
	value = strings.Join(headerNames, "\n")
	return
}

func (this *RequestHeaderNamesCheckpoint) ResponseValue(req requests.Request, resp *requests.Response, param string, options maps.Map, ruleId int64) (value any, hasRequestBody bool, sysErr error, userErr error) {
	if this.IsRequest() {
		return this.RequestValue(req, param, options, ruleId)
	}
	return
}

func (this *RequestHeaderNamesCheckpoint) CacheLife() utils.CacheLife {
	return utils.CacheShortLife
}
