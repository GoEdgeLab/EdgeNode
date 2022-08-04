package checkpoints

import (
	"bytes"
	"github.com/TeaOSLab/EdgeNode/internal/waf/requests"
	"io"
	"mime/multipart"
	"net/http"
	"testing"
)

func TestRequestUploadCheckpoint_RequestValue(t *testing.T) {
	body := bytes.NewBuffer([]byte{})

	writer := multipart.NewWriter(body)

	{
		part, err := writer.CreateFormField("name")
		if err == nil {
			_, err := part.Write([]byte("lu"))
			if err != nil {
				t.Fatal(err)
			}
		}
	}

	{
		part, err := writer.CreateFormField("age")
		if err == nil {
			_, err = part.Write([]byte("20"))
			if err != nil {
				t.Fatal(err)
			}
		}
	}

	{
		part, err := writer.CreateFormFile("myFile", "hello.txt")
		if err == nil {
			_, err = part.Write([]byte("Hello, World!"))
			if err != nil {
				t.Fatal(err)
			}
		}
	}

	{
		part, err := writer.CreateFormFile("myFile2", "hello.PHP")
		if err == nil {
			_, err = part.Write([]byte("Hello, World, PHP!"))
			if err != nil {
				t.Fatal(err)
			}
		}
	}

	{
		part, err := writer.CreateFormFile("myFile3", "hello.asp")
		if err == nil {
			_, err = part.Write([]byte("Hello, World, ASP Pages!"))
			if err != nil {
				t.Fatal(err)
			}
		}
	}

	{
		part, err := writer.CreateFormFile("myFile4", "hello.asp")
		if err == nil {
			_, err = part.Write([]byte("Hello, World, ASP Pages!"))
			if err != nil {
				t.Fatal(err)
			}
		}
	}

	err := writer.Close()
	if err != nil {
		t.Fatal(err)
	}

	rawReq, err := http.NewRequest(http.MethodPost, "http://teaos.cn/", body)
	if err != nil {
		t.Fatal()
	}

	req := requests.NewTestRequest(rawReq)
	req.WAFRaw().Header.Add("Content-Type", writer.FormDataContentType())

	checkpoint := new(RequestUploadCheckpoint)
	t.Log(checkpoint.RequestValue(req, "field", nil, 1))
	t.Log(checkpoint.RequestValue(req, "minSize", nil, 1))
	t.Log(checkpoint.RequestValue(req, "maxSize", nil, 1))
	t.Log(checkpoint.RequestValue(req, "name", nil, 1))
	t.Log(checkpoint.RequestValue(req, "ext", nil, 1))

	data, err := io.ReadAll(req.WAFRaw().Body)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(string(data))
}
