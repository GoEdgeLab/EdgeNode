package checkpoints

import (
	"bytes"
	"github.com/TeaOSLab/EdgeNode/internal/waf/requests"
	"github.com/iwind/TeaGo/types"
	"io"
	"net/http"
	"runtime"
	"strings"
	"testing"
)

func TestRequestAllCheckpoint_RequestValue(t *testing.T) {
	req, err := http.NewRequest(http.MethodPost, "http://teaos.cn/hello/world", bytes.NewBuffer([]byte("123456")))
	if err != nil {
		t.Fatal(err)
	}

	checkpoint := new(RequestAllCheckpoint)
	v, _, sysErr, userErr := checkpoint.RequestValue(requests.NewTestRequest(req), "", nil, 1)
	if sysErr != nil {
		t.Fatal(sysErr)
	}
	if userErr != nil {
		t.Fatal(userErr)
	}
	t.Log(v)
	t.Log(types.String(v))

	body, err := io.ReadAll(req.Body)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(string(body))
}

func TestRequestAllCheckpoint_RequestValue_Max(t *testing.T) {
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

func BenchmarkRequestAllCheckpoint_RequestValue(b *testing.B) {
	runtime.GOMAXPROCS(1)

	req, err := http.NewRequest(http.MethodPost, "http://teaos.cn/hello/world", bytes.NewBuffer(bytes.Repeat([]byte("HELLO"), 1024)))
	if err != nil {
		b.Fatal(err)
	}

	checkpoint := new(RequestAllCheckpoint)
	for i := 0; i < b.N; i++ {
		_, _, _, _ = checkpoint.RequestValue(requests.NewTestRequest(req), "", nil, 1)
	}
}
