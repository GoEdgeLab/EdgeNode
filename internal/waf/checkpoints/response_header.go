package checkpoints

import (
	"github.com/TeaOSLab/EdgeNode/internal/waf/requests"
	"github.com/TeaOSLab/EdgeNode/internal/waf/utils"
	"github.com/iwind/TeaGo/maps"
)

// ResponseHeaderCheckpoint ${responseHeader.arg}
type ResponseHeaderCheckpoint struct {
	Checkpoint
}

func (this *ResponseHeaderCheckpoint) IsRequest() bool {
	return false
}

func (this *ResponseHeaderCheckpoint) RequestValue(req requests.Request, param string, options maps.Map, ruleId int64) (value any, hasRequestBody bool, sysErr error, userErr error) {
	value = ""
	return
}

func (this *ResponseHeaderCheckpoint) ResponseValue(req requests.Request, resp *requests.Response, param string, options maps.Map, ruleId int64) (value any, hasRequestBody bool, sysErr error, userErr error) {
	if resp != nil && resp.Header != nil {
		value = resp.Header.Get(param)
	} else {
		value = ""
	}
	return
}

func (this *ResponseHeaderCheckpoint) CacheLife() utils.CacheLife {
	return utils.CacheMiddleLife
}
