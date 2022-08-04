package checkpoints

import (
	"bytes"
	"github.com/TeaOSLab/EdgeNode/internal/waf/requests"
	"io"
	"net/http"
	"testing"
)

func TestResponseBodyCheckpoint_ResponseValue(t *testing.T) {
	resp := requests.NewResponse(new(http.Response))
	resp.StatusCode = 200
	resp.Header = http.Header{}
	resp.Header.Set("Hello", "World")
	resp.Body = io.NopCloser(bytes.NewBuffer([]byte("Hello, World")))

	checkpoint := new(ResponseBodyCheckpoint)
	t.Log(checkpoint.ResponseValue(nil, resp, "", nil, 1))
	t.Log(checkpoint.ResponseValue(nil, resp, "", nil, 1))
	t.Log(checkpoint.ResponseValue(nil, resp, "", nil, 1))
	t.Log(checkpoint.ResponseValue(nil, resp, "", nil, 1))

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("after read:", string(data))
}
