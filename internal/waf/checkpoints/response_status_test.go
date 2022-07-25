package checkpoints

import (
	"github.com/TeaOSLab/EdgeNode/internal/waf/requests"
	"net/http"
	"testing"
)

func TestResponseStatusCheckpoint_ResponseValue(t *testing.T) {
	resp := requests.NewResponse(new(http.Response))
	resp.StatusCode = 200

	checkpoint := new(ResponseStatusCheckpoint)
	t.Log(checkpoint.ResponseValue(nil, resp, "", nil, 1))
}
