package checkpoints

import (
	"github.com/TeaOSLab/EdgeNode/internal/waf/requests"
	"github.com/iwind/TeaGo/maps"
)

type RequestHostCheckpoint struct {
	Checkpoint
}

func (this *RequestHostCheckpoint) RequestValue(req *requests.Request, param string, options maps.Map) (value interface{}, sysErr error, userErr error) {
	value = req.Host
	return
}

func (this *RequestHostCheckpoint) ResponseValue(req *requests.Request, resp *requests.Response, param string, options maps.Map) (value interface{}, sysErr error, userErr error) {
	if this.IsRequest() {
		return this.RequestValue(req, param, options)
	}
	return
}
