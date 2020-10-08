package checkpoints

import (
	"github.com/TeaOSLab/EdgeNode/internal/waf/requests"
)

type RequestContentTypeCheckpoint struct {
	Checkpoint
}

func (this *RequestContentTypeCheckpoint) RequestValue(req *requests.Request, param string, options map[string]interface{}) (value interface{}, sysErr error, userErr error) {
	value = req.Header.Get("Content-Type")
	return
}

func (this *RequestContentTypeCheckpoint) ResponseValue(req *requests.Request, resp *requests.Response, param string, options map[string]interface{}) (value interface{}, sysErr error, userErr error) {
	if this.IsRequest() {
		return this.RequestValue(req, param, options)
	}
	return
}
