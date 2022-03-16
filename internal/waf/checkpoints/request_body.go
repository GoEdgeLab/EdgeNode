package checkpoints

import (
	"github.com/TeaOSLab/EdgeNode/internal/waf/requests"
	"github.com/iwind/TeaGo/maps"
)

// RequestBodyCheckpoint ${requestBody}
type RequestBodyCheckpoint struct {
	Checkpoint
}

func (this *RequestBodyCheckpoint) RequestValue(req requests.Request, param string, options maps.Map) (value interface{}, sysErr error, userErr error) {
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
		data, err := req.WAFReadBody(int64(32 * 1024 * 1024)) // read 32m bytes
		if err != nil {
			return "", err, nil
		}

		bodyData = data
		req.WAFSetCacheBody(data)
		req.WAFRestoreBody(data)
	}

	return bodyData, nil, nil
}

func (this *RequestBodyCheckpoint) ResponseValue(req requests.Request, resp *requests.Response, param string, options maps.Map) (value interface{}, sysErr error, userErr error) {
	if this.IsRequest() {
		return this.RequestValue(req, param, options)
	}
	return
}
