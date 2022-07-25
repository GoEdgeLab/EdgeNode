package checkpoints

import (
	"github.com/TeaOSLab/EdgeNode/internal/waf/requests"
	"net/http"
	"testing"
)

func TestRequestSchemeCheckpoint_RequestValue(t *testing.T) {
	rawReq, err := http.NewRequest(http.MethodGet, "https://teaos.cn/?name=lu", nil)
	if err != nil {
		t.Fatal(err)
	}

	req := requests.NewTestRequest(rawReq)
	checkpoint := new(RequestSchemeCheckpoint)
	t.Log(checkpoint.RequestValue(req, "", nil, 1))
}
