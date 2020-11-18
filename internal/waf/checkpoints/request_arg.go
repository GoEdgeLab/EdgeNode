package checkpoints

import (
	"github.com/TeaOSLab/EdgeNode/internal/waf/requests"
	"github.com/iwind/TeaGo/maps"
)

type RequestArgCheckpoint struct {
	Checkpoint
}

func (this *RequestArgCheckpoint) RequestValue(req *requests.Request, param string, options maps.Map) (value interface{}, sysErr error, userErr error) {
	return req.URL.Query().Get(param), nil, nil
}

func (this *RequestArgCheckpoint) ResponseValue(req *requests.Request, resp *requests.Response, param string, options maps.Map) (value interface{}, sysErr error, userErr error) {
	if this.IsRequest() {
		return this.RequestValue(req, param, options)
	}
	return
}
