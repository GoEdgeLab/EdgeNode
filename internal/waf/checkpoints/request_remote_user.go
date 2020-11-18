package checkpoints

import (
	"github.com/TeaOSLab/EdgeNode/internal/waf/requests"
	"github.com/iwind/TeaGo/maps"
)

type RequestRemoteUserCheckpoint struct {
	Checkpoint
}

func (this *RequestRemoteUserCheckpoint) RequestValue(req *requests.Request, param string, options maps.Map) (value interface{}, sysErr error, userErr error) {
	username, _, ok := req.BasicAuth()
	if !ok {
		value = ""
		return
	}
	value = username
	return
}

func (this *RequestRemoteUserCheckpoint) ResponseValue(req *requests.Request, resp *requests.Response, param string, options maps.Map) (value interface{}, sysErr error, userErr error) {
	if this.IsRequest() {
		return this.RequestValue(req, param, options)
	}
	return
}
