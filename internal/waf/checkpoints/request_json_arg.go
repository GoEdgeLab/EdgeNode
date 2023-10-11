package checkpoints

import (
	"encoding/json"
	"github.com/TeaOSLab/EdgeNode/internal/utils"
	"github.com/TeaOSLab/EdgeNode/internal/waf/requests"
	wafutils "github.com/TeaOSLab/EdgeNode/internal/waf/utils"
	"github.com/iwind/TeaGo/maps"
	"strings"
)

// RequestJSONArgCheckpoint ${requestJSON.arg}
type RequestJSONArgCheckpoint struct {
	Checkpoint
}

func (this *RequestJSONArgCheckpoint) RequestValue(req requests.Request, param string, options maps.Map, ruleId int64) (value any, hasRequestBody bool, sysErr error, userErr error) {
	var bodyData = req.WAFGetCacheBody()
	hasRequestBody = true
	if len(bodyData) == 0 {
		data, err := req.WAFReadBody(req.WAFMaxRequestSize()) // read body
		if err != nil {
			return "", hasRequestBody, err, nil
		}

		bodyData = data
		req.WAFSetCacheBody(data)
		defer req.WAFRestoreBody(data)
	}

	// TODO improve performance
	var m any = nil
	err := json.Unmarshal(bodyData, &m)
	if err != nil || m == nil {
		return "", hasRequestBody, nil, err
	}

	value = utils.Get(m, strings.Split(param, "."))
	if value != nil {
		return value, hasRequestBody, nil, err
	}
	return "", hasRequestBody, nil, nil
}

func (this *RequestJSONArgCheckpoint) ResponseValue(req requests.Request, resp *requests.Response, param string, options maps.Map, ruleId int64) (value any, hasRequestBody bool, sysErr error, userErr error) {
	if this.IsRequest() {
		return this.RequestValue(req, param, options, ruleId)
	}
	return
}

func (this *RequestJSONArgCheckpoint) CacheLife() wafutils.CacheLife {
	return wafutils.CacheMiddleLife
}
