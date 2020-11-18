package checkpoints

import (
	"github.com/TeaOSLab/EdgeNode/internal/waf/requests"
	"github.com/iwind/TeaGo/maps"
	"github.com/iwind/TeaGo/types"
	"net"
)

type RequestRemotePortCheckpoint struct {
	Checkpoint
}

func (this *RequestRemotePortCheckpoint) RequestValue(req *requests.Request, param string, options maps.Map) (value interface{}, sysErr error, userErr error) {
	_, port, err := net.SplitHostPort(req.RemoteAddr)
	if err == nil {
		value = types.Int(port)
	} else {
		value = 0
	}
	return
}

func (this *RequestRemotePortCheckpoint) ResponseValue(req *requests.Request, resp *requests.Response, param string, options maps.Map) (value interface{}, sysErr error, userErr error) {
	if this.IsRequest() {
		return this.RequestValue(req, param, options)
	}
	return
}
