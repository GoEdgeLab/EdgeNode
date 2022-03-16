package checkpoints

import (
	"github.com/TeaOSLab/EdgeNode/internal/waf/requests"
	"github.com/iwind/TeaGo/maps"
)

// RequestAllCheckpoint ${requestAll}
type RequestAllCheckpoint struct {
	Checkpoint
}

func (this *RequestAllCheckpoint) RequestValue(req requests.Request, param string, options maps.Map) (value interface{}, sysErr error, userErr error) {
	valueBytes := []byte{}
	if len(req.WAFRaw().RequestURI) > 0 {
		valueBytes = append(valueBytes, req.WAFRaw().RequestURI...)
	} else if req.WAFRaw().URL != nil {
		valueBytes = append(valueBytes, req.WAFRaw().URL.RequestURI()...)
	}

	if this.RequestBodyIsEmpty(req) {
		value = valueBytes
		return
	}

	if req.WAFRaw().Body != nil {
		valueBytes = append(valueBytes, ' ')

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
		valueBytes = append(valueBytes, bodyData...)
	}

	value = valueBytes

	return
}

func (this *RequestAllCheckpoint) ResponseValue(req requests.Request, resp *requests.Response, param string, options maps.Map) (value interface{}, sysErr error, userErr error) {
	value = ""
	if this.IsRequest() {
		return this.RequestValue(req, param, options)
	}
	return
}
