package checkpoints

import (
	"github.com/TeaOSLab/EdgeNode/internal/waf/requests"
	"github.com/iwind/TeaGo/maps"
)

// ${bytesSent}
type ResponseBytesSentCheckpoint struct {
	Checkpoint
}

func (this *ResponseBytesSentCheckpoint) IsRequest() bool {
	return false
}

func (this *ResponseBytesSentCheckpoint) RequestValue(req *requests.Request, param string, options maps.Map) (value interface{}, sysErr error, userErr error) {
	value = 0
	return
}

func (this *ResponseBytesSentCheckpoint) ResponseValue(req *requests.Request, resp *requests.Response, param string, options maps.Map) (value interface{}, sysErr error, userErr error) {
	value = 0
	if resp != nil {
		value = resp.ContentLength
	}
	return
}
