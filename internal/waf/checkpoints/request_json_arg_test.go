package checkpoints

import (
	"bytes"
	"github.com/TeaOSLab/EdgeNode/internal/waf/requests"
	"io"
	"net/http"
	"testing"
)

func TestRequestJSONArgCheckpoint_RequestValue_Map(t *testing.T) {
	rawReq, err := http.NewRequest(http.MethodPost, "http://teaos.cn", bytes.NewBuffer([]byte(`
{
	"name": "lu",
	"age": 20,
	"books": [ "PHP", "Golang", "Python" ]
}
`)))
	if err != nil {
		t.Fatal(err)
	}

	req := requests.NewTestRequest(rawReq)
	//req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	checkpoint := new(RequestJSONArgCheckpoint)
	t.Log(checkpoint.RequestValue(req, "name", nil, 1))
	t.Log(checkpoint.RequestValue(req, "age", nil, 1))
	t.Log(checkpoint.RequestValue(req, "Hello", nil, 1))
	t.Log(checkpoint.RequestValue(req, "", nil, 1))
	t.Log(checkpoint.RequestValue(req, "books", nil, 1))
	t.Log(checkpoint.RequestValue(req, "books.1", nil, 1))

	body, err := io.ReadAll(req.WAFRaw().Body)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(string(body))
}

func TestRequestJSONArgCheckpoint_RequestValue_Array(t *testing.T) {
	rawReq, err := http.NewRequest(http.MethodPost, "http://teaos.cn", bytes.NewBuffer([]byte(`
[{
	"name": "lu",
	"age": 20,
	"books": [ "PHP", "Golang", "Python" ]
}]
`)))
	if err != nil {
		t.Fatal(err)
	}

	req := requests.NewTestRequest(rawReq)
	//req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	checkpoint := new(RequestJSONArgCheckpoint)
	t.Log(checkpoint.RequestValue(req, "0.name", nil, 1))
	t.Log(checkpoint.RequestValue(req, "0.age", nil, 1))
	t.Log(checkpoint.RequestValue(req, "0.Hello", nil, 1))
	t.Log(checkpoint.RequestValue(req, "", nil, 1))
	t.Log(checkpoint.RequestValue(req, "0.books", nil, 1))
	t.Log(checkpoint.RequestValue(req, "0.books.1", nil, 1))

	body, err := io.ReadAll(req.WAFRaw().Body)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(string(body))
}

func TestRequestJSONArgCheckpoint_RequestValue_Error(t *testing.T) {
	rawReq, err := http.NewRequest(http.MethodPost, "http://teaos.cn", bytes.NewBuffer([]byte(`
[{
	"name": "lu",
	"age": 20,
	"books": [ "PHP", "Golang", "Python" ]
}]
`)))
	if err != nil {
		t.Fatal(err)
	}

	req := requests.NewTestRequest(rawReq)
	//req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	checkpoint := new(RequestJSONArgCheckpoint)
	t.Log(checkpoint.RequestValue(req, "0.name", nil, 1))
	t.Log(checkpoint.RequestValue(req, "0.age", nil, 1))
	t.Log(checkpoint.RequestValue(req, "0.Hello", nil, 1))
	t.Log(checkpoint.RequestValue(req, "", nil, 1))
	t.Log(checkpoint.RequestValue(req, "0.books", nil, 1))
	t.Log(checkpoint.RequestValue(req, "0.books.1", nil, 1))

	body, err := io.ReadAll(req.WAFRaw().Body)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(string(body))
}
