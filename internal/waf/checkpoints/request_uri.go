package checkpoints

import (
	"github.com/TeaOSLab/EdgeNode/internal/waf/requests"
	"github.com/iwind/TeaGo/maps"
)

type RequestURICheckpoint struct {
	Checkpoint
}

func (this *RequestURICheckpoint) RequestValue(req *requests.Request, param string, options maps.Map) (value interface{}, sysErr error, userErr error) {
	if len(req.RequestURI) > 0 {
		value = req.RequestURI
	} else if req.URL != nil {
		value = req.URL.RequestURI()
	}
	return
}

func (this *RequestURICheckpoint) ResponseValue(req *requests.Request, resp *requests.Response, param string, options maps.Map) (value interface{}, sysErr error, userErr error) {
	if this.IsRequest() {
		return this.RequestValue(req, param, options)
	}
	return
}
