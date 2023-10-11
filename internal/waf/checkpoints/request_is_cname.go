package checkpoints

import (
	"github.com/TeaOSLab/EdgeNode/internal/waf/requests"
	"github.com/TeaOSLab/EdgeNode/internal/waf/utils"
	"github.com/iwind/TeaGo/maps"
)

type RequestIsCNAMECheckpoint struct {
	Checkpoint
}

func (this *RequestIsCNAMECheckpoint) RequestValue(req requests.Request, param string, options maps.Map, ruleId int64) (value any, hasRequestBody bool, sysErr error, userErr error) {
	if req.Format("${cname}") == req.Format("${host}") {
		value = 1
	} else {
		value = 0
	}
	return
}

func (this *RequestIsCNAMECheckpoint) ResponseValue(req requests.Request, resp *requests.Response, param string, options maps.Map, ruleId int64) (value any, hasRequestBody bool, sysErr error, userErr error) {
	if this.IsRequest() {
		return this.RequestValue(req, param, options, ruleId)
	}
	return
}

func (this *RequestIsCNAMECheckpoint) CacheLife() utils.CacheLife {
	return utils.CacheLongLife
}
