package checkpoints

import (
	"github.com/TeaOSLab/EdgeNode/internal/waf/requests"
	"github.com/TeaOSLab/EdgeNode/internal/waf/utils"
	"github.com/iwind/TeaGo/maps"
)

// RequestBodyCheckpoint ${requestBody}
type RequestBodyCheckpoint struct {
	Checkpoint
}

func (this *RequestBodyCheckpoint) RequestValue(req requests.Request, param string, options maps.Map) (value interface{}, hasRequestBody bool, sysErr error, userErr error) {
	if this.RequestBodyIsEmpty(req) {
		value = ""
		return
	}

	if req.WAFRaw().Body == nil {
		value = ""
		return
	}

	var bodyData = req.WAFGetCacheBody()
	hasRequestBody = true
	if len(bodyData) == 0 {
		data, err := req.WAFReadBody(utils.MaxBodySize) // read body
		if err != nil {
			return "", hasRequestBody, err, nil
		}

		bodyData = data
		req.WAFSetCacheBody(data)
		req.WAFRestoreBody(data)
	}

	return bodyData, hasRequestBody, nil, nil
}

func (this *RequestBodyCheckpoint) ResponseValue(req requests.Request, resp *requests.Response, param string, options maps.Map) (value interface{}, hasRequestBody bool, sysErr error, userErr error) {
	if this.IsRequest() {
		return this.RequestValue(req, param, options)
	}
	return
}
