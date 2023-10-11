package checkpoints

import (
	"github.com/TeaOSLab/EdgeNode/internal/waf/requests"
	"github.com/TeaOSLab/EdgeNode/internal/waf/utils"
	"github.com/iwind/TeaGo/maps"
	"net/url"
)

// RequestFormArgCheckpoint ${requestForm.arg}
type RequestFormArgCheckpoint struct {
	Checkpoint
}

func (this *RequestFormArgCheckpoint) RequestValue(req requests.Request, param string, options maps.Map, ruleId int64) (value any, hasRequestBody bool, sysErr error, userErr error) {
	hasRequestBody = true

	if this.RequestBodyIsEmpty(req) {
		value = ""
		return
	}

	if req.WAFRaw().Body == nil {
		value = ""
		return
	}

	var bodyData = req.WAFGetCacheBody()
	if len(bodyData) == 0 {
		data, err := req.WAFReadBody(req.WAFMaxRequestSize()) // read body
		if err != nil {
			return "", hasRequestBody, err, nil
		}

		bodyData = data
		req.WAFSetCacheBody(data)
		req.WAFRestoreBody(data)
	}

	// TODO improve performance
	values, _ := url.ParseQuery(string(bodyData))
	return values.Get(param), hasRequestBody, nil, nil
}

func (this *RequestFormArgCheckpoint) ResponseValue(req requests.Request, resp *requests.Response, param string, options maps.Map, ruleId int64) (value any, hasRequestBody bool, sysErr error, userErr error) {
	if this.IsRequest() {
		return this.RequestValue(req, param, options, ruleId)
	}
	return
}

func (this *RequestFormArgCheckpoint) CacheLife() utils.CacheLife {
	return utils.CacheMiddleLife
}
