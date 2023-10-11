package checkpoints

import (
	"github.com/TeaOSLab/EdgeNode/internal/waf/requests"
	"github.com/TeaOSLab/EdgeNode/internal/waf/utils"
	"github.com/iwind/TeaGo/maps"
	"github.com/iwind/TeaGo/types"
)

type RequestGeneralHeaderLengthCheckpoint struct {
	Checkpoint
}

func (this *RequestGeneralHeaderLengthCheckpoint) IsComposed() bool {
	return true
}

func (this *RequestGeneralHeaderLengthCheckpoint) RequestValue(req requests.Request, param string, options maps.Map, ruleId int64) (value any, hasRequestBody bool, sysErr error, userErr error) {
	value = false

	var headers = options.GetSlice("headers")
	if len(headers) == 0 {
		return
	}

	var length = options.GetInt("length")

	for _, header := range headers {
		v := req.WAFRaw().Header.Get(types.String(header))
		if len(v) > length {
			value = true
			break
		}
	}

	return
}

func (this *RequestGeneralHeaderLengthCheckpoint) ResponseValue(req requests.Request, resp *requests.Response, param string, options maps.Map, ruleId int64) (value any, hasRequestBody bool, sysErr error, userErr error) {
	return
}

func (this *RequestGeneralHeaderLengthCheckpoint) CacheLife() utils.CacheLife {
	return utils.CacheDisabled
}
