package checkpoints

import (
	"bytes"
	"github.com/TeaOSLab/EdgeNode/internal/waf/requests"
	"io"
	"net/http"
	"net/url"
	"testing"
)

func TestRequestFormArgCheckpoint_RequestValue(t *testing.T) {
	rawReq, err := http.NewRequest(http.MethodPost, "http://teaos.cn", bytes.NewBuffer([]byte("name=lu&age=20&encoded="+url.QueryEscape("<strong>ENCODED STRING</strong>"))))
	if err != nil {
		t.Fatal(err)
	}

	req := requests.NewTestRequest(rawReq)
	req.WAFRaw().Header.Set("Content-Type", "application/x-www-form-urlencoded")

	checkpoint := new(RequestFormArgCheckpoint)
	t.Log(checkpoint.RequestValue(req, "name", nil, 1))
	t.Log(checkpoint.RequestValue(req, "age", nil, 1))
	t.Log(checkpoint.RequestValue(req, "Hello", nil, 1))
	t.Log(checkpoint.RequestValue(req, "encoded", nil, 1))

	body, err := io.ReadAll(req.WAFRaw().Body)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(string(body))
}
