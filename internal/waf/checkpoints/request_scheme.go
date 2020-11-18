package checkpoints

import (
	"github.com/TeaOSLab/EdgeNode/internal/waf/requests"
	"github.com/iwind/TeaGo/maps"
)

type RequestSchemeCheckpoint struct {
	Checkpoint
}

func (this *RequestSchemeCheckpoint) RequestValue(req *requests.Request, param string, options maps.Map) (value interface{}, sysErr error, userErr error) {
	value = req.URL.Scheme
	return
}

func (this *RequestSchemeCheckpoint) ResponseValue(req *requests.Request, resp *requests.Response, param string, options maps.Map) (value interface{}, sysErr error, userErr error) {
	if this.IsRequest() {
		return this.RequestValue(req, param, options)
	}
	return
}
