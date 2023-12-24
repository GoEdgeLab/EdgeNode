package checkpoints_test

import (
	"github.com/TeaOSLab/EdgeNode/internal/waf/checkpoints"
	"github.com/TeaOSLab/EdgeNode/internal/waf/requests"
	"net/http"
	"testing"
)

func TestRequestRefererOriginCheckpoint_RequestValue(t *testing.T) {
	rawReq, err := http.NewRequest(http.MethodGet, "https://example.com", nil)
	if err != nil {
		t.Fatal(err)
	}
	var req = requests.NewTestRequest(rawReq)

	var checkpoint = &checkpoints.RequestRefererOriginCheckpoint{}

	{
		t.Log(checkpoint.RequestValue(req, "", nil, 0))
	}

	{
		rawReq.Header.Set("Referer", "https://example.com/hello.yaml")
		t.Log(checkpoint.RequestValue(req, "", nil, 0))
	}

	{
		rawReq.Header.Set("Origin", "https://example.com/world.yaml")
		t.Log(checkpoint.RequestValue(req, "", nil, 0))
	}

	{
		rawReq.Header.Del("Referer")
		rawReq.Header.Set("Origin", "https://example.com/world.yaml")
		t.Log(checkpoint.RequestValue(req, "", nil, 0))
	}
}
