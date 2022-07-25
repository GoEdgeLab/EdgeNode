package checkpoints

import (
	"github.com/TeaOSLab/EdgeNode/internal/waf/requests"
	"github.com/iwind/TeaGo/maps"
	"net/http"
	"testing"
)

func TestCCCheckpoint_RequestValue(t *testing.T) {
	raw, err := http.NewRequest(http.MethodGet, "http://teaos.cn/", nil)
	if err != nil {
		t.Fatal(err)
	}

	req := requests.NewTestRequest(raw)
	req.WAFRaw().RemoteAddr = "127.0.0.1"

	checkpoint := new(CCCheckpoint)
	checkpoint.Init()
	checkpoint.Start()

	options := maps.Map{
		"period": "5",
	}
	t.Log(checkpoint.RequestValue(req, "requests", options, 1))
	t.Log(checkpoint.RequestValue(req, "requests", options, 1))

	req.WAFRaw().RemoteAddr = "127.0.0.2"
	t.Log(checkpoint.RequestValue(req, "requests", options, 1))

	req.WAFRaw().RemoteAddr = "127.0.0.1"
	t.Log(checkpoint.RequestValue(req, "requests", options, 1))

	req.WAFRaw().RemoteAddr = "127.0.0.2"
	t.Log(checkpoint.RequestValue(req, "requests", options, 1))

	req.WAFRaw().RemoteAddr = "127.0.0.2"
	t.Log(checkpoint.RequestValue(req, "requests", options, 1))

	req.WAFRaw().RemoteAddr = "127.0.0.2"
	t.Log(checkpoint.RequestValue(req, "requests", options, 1))
}
