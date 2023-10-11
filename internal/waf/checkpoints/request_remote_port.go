package checkpoints

import (
	"github.com/TeaOSLab/EdgeNode/internal/waf/requests"
	"github.com/TeaOSLab/EdgeNode/internal/waf/utils"
	"github.com/iwind/TeaGo/maps"
	"github.com/iwind/TeaGo/types"
	"net"
)

type RequestRemotePortCheckpoint struct {
	Checkpoint
}

func (this *RequestRemotePortCheckpoint) RequestValue(req requests.Request, param string, options maps.Map, ruleId int64) (value any, hasRequestBody bool, sysErr error, userErr error) {
	_, port, err := net.SplitHostPort(req.WAFRaw().RemoteAddr)
	if err == nil {
		value = types.Int(port)
	} else {
		value = 0
	}
	return
}

func (this *RequestRemotePortCheckpoint) ResponseValue(req requests.Request, resp *requests.Response, param string, options maps.Map, ruleId int64) (value any, hasRequestBody bool, sysErr error, userErr error) {
	if this.IsRequest() {
		return this.RequestValue(req, param, options, ruleId)
	}
	return
}

func (this *RequestRemotePortCheckpoint) CacheLife() utils.CacheLife {
	return utils.CacheMiddleLife
}
