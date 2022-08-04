package checkpoints

import (
	"bytes"
	"github.com/TeaOSLab/EdgeNode/internal/waf/requests"
	"github.com/iwind/TeaGo/types"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestRequestBodyCheckpoint_RequestValue(t *testing.T) {
	rawReq, err := http.NewRequest(http.MethodPost, "http://teaos.cn", bytes.NewBuffer([]byte("123456")))
	if err != nil {
		t.Fatal(err)
	}
	var req = requests.NewTestRequest(rawReq)
	checkpoint := new(RequestBodyCheckpoint)
	t.Log(checkpoint.RequestValue(req, "", nil, 1))

	body, err := io.ReadAll(rawReq.Body)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(string(body))
	t.Log(string(req.WAFGetCacheBody()))
}

func TestRequestBodyCheckpoint_RequestValue_Max(t *testing.T) {
	req, err := http.NewRequest(http.MethodPost, "http://teaos.cn", bytes.NewBuffer([]byte(strings.Repeat("123456", 10240000))))
	if err != nil {
		t.Fatal(err)
	}

	checkpoint := new(RequestBodyCheckpoint)
	value, _, err, _ := checkpoint.RequestValue(requests.NewTestRequest(req), "", nil, 1)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("value bytes:", len(types.String(value)))

	body, err := io.ReadAll(req.Body)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("raw bytes:", len(body))
}
