package checkpoints

import (
	"github.com/TeaOSLab/EdgeNode/internal/waf/requests"
)

type RequestArgsCheckpoint struct {
	Checkpoint
}

func (this *RequestArgsCheckpoint) RequestValue(req *requests.Request, param string, options map[string]interface{}) (value interface{}, sysErr error, userErr error) {
	value = req.URL.RawQuery
	return
}

func (this *RequestArgsCheckpoint) ResponseValue(req *requests.Request, resp *requests.Response, param string, options map[string]interface{}) (value interface{}, sysErr error, userErr error) {
	if this.IsRequest() {
		return this.RequestValue(req, param, options)
	}
	return
}
