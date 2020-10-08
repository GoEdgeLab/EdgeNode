package checkpoints

import (
	"github.com/TeaOSLab/EdgeNode/internal/waf/requests"
	"net"
)

type RequestRawRemoteAddrCheckpoint struct {
	Checkpoint
}

func (this *RequestRawRemoteAddrCheckpoint) RequestValue(req *requests.Request, param string, options map[string]interface{}) (value interface{}, sysErr error, userErr error) {
	host, _, err := net.SplitHostPort(req.RemoteAddr)
	if err == nil {
		value = host
	} else {
		value = req.RemoteAddr
	}
	return
}

func (this *RequestRawRemoteAddrCheckpoint) ResponseValue(req *requests.Request, resp *requests.Response, param string, options map[string]interface{}) (value interface{}, sysErr error, userErr error) {
	if this.IsRequest() {
		return this.RequestValue(req, param, options)
	}
	return
}
