package checkpoints

import (
	"bytes"
	"github.com/TeaOSLab/EdgeNode/internal/waf/requests"
	"github.com/TeaOSLab/EdgeNode/internal/waf/utils"
	"github.com/iwind/TeaGo/maps"
	"io"
)

// ResponseBodyCheckpoint ${responseBody}
type ResponseBodyCheckpoint struct {
	Checkpoint
}

func (this *ResponseBodyCheckpoint) IsRequest() bool {
	return false
}

func (this *ResponseBodyCheckpoint) RequestValue(req requests.Request, param string, options maps.Map, ruleId int64) (value any, hasRequestBody bool, sysErr error, userErr error) {
	value = ""
	return
}

func (this *ResponseBodyCheckpoint) ResponseValue(req requests.Request, resp *requests.Response, param string, options maps.Map, ruleId int64) (value any, hasRequestBody bool, sysErr error, userErr error) {
	if resp.ContentLength == 0 {
		value = ""
		return
	}

	value = ""
	if resp != nil && resp.Body != nil {
		if len(resp.BodyData) > 0 {
			value = string(resp.BodyData)
			return
		}
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			sysErr = err
			return
		}
		resp.BodyData = body
		_ = resp.Body.Close()
		value = body
		resp.Body = io.NopCloser(bytes.NewBuffer(body))
	}
	return
}

func (this *ResponseBodyCheckpoint) CacheLife() utils.CacheLife {
	return utils.CacheMiddleLife
}
