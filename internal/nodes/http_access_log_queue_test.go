// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package nodes

import (
	"bytes"
	"github.com/TeaOSLab/EdgeCommon/pkg/rpc/pb"
	"github.com/TeaOSLab/EdgeNode/internal/rpc"
	_ "github.com/iwind/TeaGo/bootstrap"
	"google.golang.org/grpc/status"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestHTTPAccessLogQueue_Push(t *testing.T) {
	// 发送到API
	client, err := rpc.SharedRPC()
	if err != nil {
		t.Fatal(err)
	}

	var requestId = 1_000_000

	var utf8Bytes = []byte{}
	for i := 0; i < 254; i++ {
		utf8Bytes = append(utf8Bytes, uint8(i))
	}

	//bytes = []byte("真不错")

	//t.Log(strings.ToValidUTF8(string(utf8Bytes), ""))
	_, err = client.HTTPAccessLogRPC().CreateHTTPAccessLogs(client.Context(), &pb.CreateHTTPAccessLogsRequest{HttpAccessLogs: []*pb.HTTPAccessLog{
		{
			ServerId:    23,
			RequestId:   strconv.FormatInt(time.Now().Unix(), 10) + strconv.Itoa(requestId) + strconv.FormatInt(1, 10),
			NodeId:      48,
			Host:        "www.hello.com",
			RequestURI:  string(utf8Bytes),
			RequestPath: string(utf8Bytes),
			Timestamp:   time.Now().Unix(),
		},
	}})
	if err != nil {
		// 这里只是为了重现错误
		t.Logf("%#v, %s", err, err.Error())

		statusErr, ok := status.FromError(err)
		if ok {
			t.Logf("%#v", statusErr)
		}
		return
	}
	t.Log("ok")
}

func TestHTTPAccessLogQueue_Push2(t *testing.T) {
	var utf8Bytes = []byte{}
	for i := 0; i < 254; i++ {
		utf8Bytes = append(utf8Bytes, uint8(i))
	}

	var accessLog = &pb.HTTPAccessLog{
		ServerId:    23,
		RequestId:   strconv.FormatInt(time.Now().Unix(), 10) + strconv.Itoa(1) + strconv.FormatInt(1, 10),
		NodeId:      48,
		Host:        "www.hello.com",
		RequestURI:  string(utf8Bytes),
		RequestPath: string(utf8Bytes),
		Timestamp:   time.Now().Unix(),
	}
	var v = reflect.Indirect(reflect.ValueOf(accessLog))
	var countFields = v.NumField()
	for i := 0; i < countFields; i++ {
		var field = v.Field(i)
		if field.Kind() == reflect.String {
			field.SetString(strings.ToValidUTF8(field.String(), ""))
		}
	}

	client, err := rpc.SharedRPC()
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.HTTPAccessLogRPC().CreateHTTPAccessLogs(client.Context(), &pb.CreateHTTPAccessLogsRequest{HttpAccessLogs: []*pb.HTTPAccessLog{
		accessLog,
	}})
	if err != nil {
		t.Fatal(err)
	}
	t.Log("ok")
}

func BenchmarkHTTPAccessLogQueue_ToValidUTF8(b *testing.B) {
	runtime.GOMAXPROCS(1)

	var utf8Bytes = []byte{}
	for i := 0; i < 254; i++ {
		utf8Bytes = append(utf8Bytes, uint8(i))
	}

	for i := 0; i < b.N; i++ {
		_ = bytes.ToValidUTF8(utf8Bytes, nil)
	}
}

func BenchmarkHTTPAccessLogQueue_ToValidUTF8String(b *testing.B) {
	runtime.GOMAXPROCS(1)

	var utf8Bytes = []byte{}
	for i := 0; i < 254; i++ {
		utf8Bytes = append(utf8Bytes, uint8(i))
	}

	var s = string(utf8Bytes)
	for i := 0; i < b.N; i++ {
		_ = strings.ToValidUTF8(s, "")
	}
}
