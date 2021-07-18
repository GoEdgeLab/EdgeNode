package checkpoints

import (
	"encoding/json"
	"github.com/TeaOSLab/EdgeNode/internal/utils"
	"github.com/TeaOSLab/EdgeNode/internal/waf/requests"
	"github.com/iwind/TeaGo/maps"
	"strings"
)

// RequestJSONArgCheckpoint ${requestJSON.arg}
type RequestJSONArgCheckpoint struct {
	Checkpoint
}

func (this *RequestJSONArgCheckpoint) RequestValue(req requests.Request, param string, options maps.Map) (value interface{}, sysErr error, userErr error) {
	var bodyData = req.WAFGetCacheBody()
	if len(bodyData) == 0 {
		data, err := req.WAFReadBody(int64(32 * 1024 * 1024)) // read 32m bytes
		if err != nil {
			return "", err, nil
		}

		bodyData = data
		req.WAFSetCacheBody(data)
		defer req.WAFRestoreBody(data)
	}

	// TODO improve performance
	var m interface{} = nil
	err := json.Unmarshal(bodyData, &m)
	if err != nil || m == nil {
		return "", nil, err
	}

	value = utils.Get(m, strings.Split(param, "."))
	if value != nil {
		return value, nil, err
	}
	return "", nil, nil
}

func (this *RequestJSONArgCheckpoint) ResponseValue(req requests.Request, resp *requests.Response, param string, options maps.Map) (value interface{}, sysErr error, userErr error) {
	if this.IsRequest() {
		return this.RequestValue(req, param, options)
	}
	return
}
