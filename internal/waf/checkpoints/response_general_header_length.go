package checkpoints

import (
	"github.com/TeaOSLab/EdgeNode/internal/waf/requests"
	"github.com/TeaOSLab/EdgeNode/internal/waf/utils"
	"github.com/iwind/TeaGo/maps"
	"github.com/iwind/TeaGo/types"
)

type ResponseGeneralHeaderLengthCheckpoint struct {
	Checkpoint
}

func (this *ResponseGeneralHeaderLengthCheckpoint) IsRequest() bool {
	return false
}

func (this *ResponseGeneralHeaderLengthCheckpoint) IsComposed() bool {
	return true
}

func (this *ResponseGeneralHeaderLengthCheckpoint) RequestValue(req requests.Request, param string, options maps.Map, ruleId int64) (value any, hasRequestBody bool, sysErr error, userErr error) {
	return
}

func (this *ResponseGeneralHeaderLengthCheckpoint) ResponseValue(req requests.Request, resp *requests.Response, param string, options maps.Map, ruleId int64) (value any, hasRequestBody bool, sysErr error, userErr error) {
	value = false

	headers := options.GetSlice("headers")
	if len(headers) == 0 {
		return
	}

	length := options.GetInt("length")

	for _, header := range headers {
		v := req.WAFRaw().Header.Get(types.String(header))
		if len(v) > length {
			value = true
			break
		}
	}

	return
}

func (this *ResponseGeneralHeaderLengthCheckpoint) CacheLife() utils.CacheLife {
	return utils.CacheMiddleLife
}
