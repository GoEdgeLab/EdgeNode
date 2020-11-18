package checkpoints

import (
	"github.com/TeaOSLab/EdgeNode/internal/waf/requests"
	"github.com/iwind/TeaGo/maps"
	"net"
	"strings"
)

type RequestRemoteAddrCheckpoint struct {
	Checkpoint
}

func (this *RequestRemoteAddrCheckpoint) RequestValue(req *requests.Request, param string, options maps.Map) (value interface{}, sysErr error, userErr error) {
	// X-Forwarded-For
	forwardedFor := req.Header.Get("X-Forwarded-For")
	if len(forwardedFor) > 0 {
		commaIndex := strings.Index(forwardedFor, ",")
		if commaIndex > 0 {
			value = forwardedFor[:commaIndex]
			return
		}
		value = forwardedFor
		return
	}

	// Real-IP
	{
		realIP, ok := req.Header["X-Real-IP"]
		if ok && len(realIP) > 0 {
			value = realIP[0]
			return
		}
	}

	// Real-Ip
	{
		realIP, ok := req.Header["X-Real-Ip"]
		if ok && len(realIP) > 0 {
			value = realIP[0]
			return
		}
	}

	// Remote-Addr
	host, _, err := net.SplitHostPort(req.RemoteAddr)
	if err == nil {
		value = host
	} else {
		value = req.RemoteAddr
	}
	return
}

func (this *RequestRemoteAddrCheckpoint) ResponseValue(req *requests.Request, resp *requests.Response, param string, options maps.Map) (value interface{}, sysErr error, userErr error) {
	if this.IsRequest() {
		return this.RequestValue(req, param, options)
	}
	return
}
