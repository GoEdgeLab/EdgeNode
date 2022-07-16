package checkpoints

import (
	"github.com/TeaOSLab/EdgeNode/internal/waf/requests"
	"github.com/iwind/TeaGo/maps"
)

type RequestURLCheckpoint struct {
	Checkpoint
}

func (this *RequestURLCheckpoint) RequestValue(req requests.Request, param string, options maps.Map) (value interface{}, hasRequestBody bool, sysErr error, userErr error) {
	return req.Format("${requestURL}"), hasRequestBody, nil, nil
}

func (this *RequestURLCheckpoint) ResponseValue(req requests.Request, resp *requests.Response, param string, options maps.Map) (value interface{}, hasRequestBody bool, sysErr error, userErr error) {
	if this.IsRequest() {
		return this.RequestValue(req, param, options)
	}
	return
}
