package checkpoints

import (
	"github.com/TeaOSLab/EdgeNode/internal/waf/requests"
	"github.com/iwind/TeaGo/maps"
)

type RequestURICheckpoint struct {
	Checkpoint
}

func (this *RequestURICheckpoint) RequestValue(req requests.Request, param string, options maps.Map, ruleId int64) (value interface{}, hasRequestBody bool, sysErr error, userErr error) {
	if len(req.WAFRaw().RequestURI) > 0 {
		value = req.WAFRaw().RequestURI
	} else if req.WAFRaw().URL != nil {
		value = req.WAFRaw().URL.RequestURI()
	}
	return
}

func (this *RequestURICheckpoint) ResponseValue(req requests.Request, resp *requests.Response, param string, options maps.Map, ruleId int64) (value interface{}, hasRequestBody bool, sysErr error, userErr error) {
	if this.IsRequest() {
		return this.RequestValue(req, param, options, ruleId)
	}
	return
}
