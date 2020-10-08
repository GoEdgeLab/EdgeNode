package checkpoints

import (
	"github.com/TeaOSLab/EdgeNode/internal/waf/requests"
)

type RequestCookieCheckpoint struct {
	Checkpoint
}

func (this *RequestCookieCheckpoint) RequestValue(req *requests.Request, param string, options map[string]interface{}) (value interface{}, sysErr error, userErr error) {
	cookie, err := req.Cookie(param)
	if err != nil {
		value = ""
		return
	}

	value = cookie.Value
	return
}

func (this *RequestCookieCheckpoint) ResponseValue(req *requests.Request, resp *requests.Response, param string, options map[string]interface{}) (value interface{}, sysErr error, userErr error) {
	if this.IsRequest() {
		return this.RequestValue(req, param, options)
	}
	return
}
