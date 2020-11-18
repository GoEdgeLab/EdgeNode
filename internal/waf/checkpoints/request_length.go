package checkpoints

import (
	"github.com/TeaOSLab/EdgeNode/internal/waf/requests"
	"github.com/iwind/TeaGo/maps"
)

type RequestLengthCheckpoint struct {
	Checkpoint
}

func (this *RequestLengthCheckpoint) RequestValue(req *requests.Request, param string, options maps.Map) (value interface{}, sysErr error, userErr error) {
	value = req.ContentLength
	return
}

func (this *RequestLengthCheckpoint) ResponseValue(req *requests.Request, resp *requests.Response, param string, options maps.Map) (value interface{}, sysErr error, userErr error) {
	if this.IsRequest() {
		return this.RequestValue(req, param, options)
	}
	return
}
