package checkpoints

import (
	"github.com/TeaOSLab/EdgeNode/internal/waf/requests"
	"net/http"
	"testing"
)

func TestRequestHostCheckpoint_RequestValue(t *testing.T) {
	rawReq, err := http.NewRequest(http.MethodGet, "https://teaos.cn/?name=lu", nil)
	if err != nil {
		t.Fatal(err)
	}

	req := requests.NewTestRequest(rawReq)
	req.WAFRaw().Header.Set("Host", "cloud.teaos.cn")

	checkpoint := new(RequestHostCheckpoint)
	t.Log(checkpoint.RequestValue(req, "", nil, 1))
}
