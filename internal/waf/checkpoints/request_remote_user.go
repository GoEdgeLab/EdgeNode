package checkpoints

import (
	"github.com/TeaOSLab/EdgeNode/internal/waf/requests"
)

type RequestRemoteUserCheckpoint struct {
	Checkpoint
}

func (this *RequestRemoteUserCheckpoint) RequestValue(req *requests.Request, param string, options map[string]interface{}) (value interface{}, sysErr error, userErr error) {
	username, _, ok := req.BasicAuth()
	if !ok {
		value = ""
		return
	}
	value = username
	return
}

func (this *RequestRemoteUserCheckpoint) ResponseValue(req *requests.Request, resp *requests.Response, param string, options map[string]interface{}) (value interface{}, sysErr error, userErr error) {
	if this.IsRequest() {
		return this.RequestValue(req, param, options)
	}
	return
}
