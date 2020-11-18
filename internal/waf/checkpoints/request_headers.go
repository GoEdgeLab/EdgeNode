package checkpoints

import (
	"github.com/TeaOSLab/EdgeNode/internal/waf/requests"
	"github.com/iwind/TeaGo/maps"
	"sort"
	"strings"
)

type RequestHeadersCheckpoint struct {
	Checkpoint
}

func (this *RequestHeadersCheckpoint) RequestValue(req *requests.Request, param string, options maps.Map) (value interface{}, sysErr error, userErr error) {
	var headers = []string{}
	for k, v := range req.Header {
		for _, subV := range v {
			headers = append(headers, k+": "+subV)
		}
	}
	sort.Strings(headers)
	value = strings.Join(headers, "\n")
	return
}

func (this *RequestHeadersCheckpoint) ResponseValue(req *requests.Request, resp *requests.Response, param string, options maps.Map) (value interface{}, sysErr error, userErr error) {
	if this.IsRequest() {
		return this.RequestValue(req, param, options)
	}
	return
}
