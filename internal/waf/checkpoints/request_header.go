package checkpoints

import (
	"github.com/TeaOSLab/EdgeNode/internal/waf/requests"
	"strings"
)

type RequestHeaderCheckpoint struct {
	Checkpoint
}

func (this *RequestHeaderCheckpoint) RequestValue(req *requests.Request, param string, options map[string]interface{}) (value interface{}, sysErr error, userErr error) {
	v, found := req.Header[param]
	if !found {
		value = ""
		return
	}
	value = strings.Join(v, ";")
	return
}

func (this *RequestHeaderCheckpoint) ResponseValue(req *requests.Request, resp *requests.Response, param string, options map[string]interface{}) (value interface{}, sysErr error, userErr error) {
	if this.IsRequest() {
		return this.RequestValue(req, param, options)
	}
	return
}
