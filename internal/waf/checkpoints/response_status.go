package checkpoints

import (
	"github.com/TeaOSLab/EdgeNode/internal/waf/requests"
	"github.com/iwind/TeaGo/maps"
)

// ${bytesSent}
type ResponseStatusCheckpoint struct {
	Checkpoint
}

func (this *ResponseStatusCheckpoint) IsRequest() bool {
	return false
}

func (this *ResponseStatusCheckpoint) RequestValue(req *requests.Request, param string, options maps.Map) (value interface{}, sysErr error, userErr error) {
	value = 0
	return
}

func (this *ResponseStatusCheckpoint) ResponseValue(req *requests.Request, resp *requests.Response, param string, options maps.Map) (value interface{}, sysErr error, userErr error) {
	if resp != nil {
		value = resp.StatusCode
	}
	return
}
