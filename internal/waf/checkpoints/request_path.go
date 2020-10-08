package checkpoints

import (
	"github.com/TeaOSLab/EdgeNode/internal/waf/requests"
)

type RequestPathCheckpoint struct {
	Checkpoint
}

func (this *RequestPathCheckpoint) RequestValue(req *requests.Request, param string, options map[string]interface{}) (value interface{}, sysErr error, userErr error) {
	return req.URL.Path, nil, nil
}

func (this *RequestPathCheckpoint) ResponseValue(req *requests.Request, resp *requests.Response, param string, options map[string]interface{}) (value interface{}, sysErr error, userErr error) {
	if this.IsRequest() {
		return this.RequestValue(req, param, options)
	}
	return
}
