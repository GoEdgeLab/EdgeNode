package nodes

import (
	teaconst "github.com/TeaOSLab/EdgeNode/internal/const"
	"github.com/TeaOSLab/EdgeNode/internal/zero"
	"github.com/iwind/TeaGo/assert"
	"runtime"
	"sync"
	"testing"
	"time"
)

func TestHTTPRequest_httpRequestGenBoundary(t *testing.T) {
	for i := 0; i < 10; i++ {
		var boundary = httpRequestGenBoundary()
		t.Log(boundary, "[", len(boundary), "bytes", "]")
	}
}

func TestHTTPRequest_httpRequestParseBoundary(t *testing.T) {
	var a = assert.NewAssertion(t)
	a.IsTrue(httpRequestParseBoundary("multipart/byteranges") == "")
	a.IsTrue(httpRequestParseBoundary("multipart/byteranges; boundary=123") == "123")
	a.IsTrue(httpRequestParseBoundary("multipart/byteranges; boundary=123; 456") == "123")
}

func TestHTTPRequest_httpRequestParseRangeHeader(t *testing.T) {
	var a = assert.NewAssertion(t)
	{
		_, ok := httpRequestParseRangeHeader("")
		a.IsFalse(ok)
	}
	{
		_, ok := httpRequestParseRangeHeader("byte=")
		a.IsFalse(ok)
	}
	{
		_, ok := httpRequestParseRangeHeader("byte=")
		a.IsFalse(ok)
	}
	{
		set, ok := httpRequestParseRangeHeader("bytes=")
		a.IsTrue(ok)
		a.IsTrue(len(set) == 0)
	}
	{
		_, ok := httpRequestParseRangeHeader("bytes=60-50")
		a.IsFalse(ok)
	}
	{
		set, ok := httpRequestParseRangeHeader("bytes=0-50")
		a.IsTrue(ok)
		a.IsTrue(len(set) > 0)
		t.Log(set)
	}
	{
		set, ok := httpRequestParseRangeHeader("bytes=0-")
		a.IsTrue(ok)
		a.IsTrue(len(set) > 0)
		if len(set) > 0 {
			a.IsTrue(set[0][0] == 0)
		}
		t.Log(set)
	}
	{
		set, ok := httpRequestParseRangeHeader("bytes=-50")
		a.IsTrue(ok)
		a.IsTrue(len(set) > 0)
		t.Log(set)
	}
	{
		set, ok := httpRequestParseRangeHeader("bytes=0-50, 60-100")
		a.IsTrue(ok)
		a.IsTrue(len(set) > 0)
		t.Log(set)
	}
}

func TestHTTPRequest_httpRequestParseContentRangeHeader(t *testing.T) {
	{
		var c1 = "bytes 0-100/*"
		t.Log(httpRequestParseContentRangeHeader(c1))
	}
	{
		var c1 = "bytes 30-100/*"
		t.Log(httpRequestParseContentRangeHeader(c1))
	}
	{
		var c1 = "bytes1 0-100/*"
		t.Log(httpRequestParseContentRangeHeader(c1))
	}
}

func BenchmarkHTTPRequest_httpRequestParseContentRangeHeader(b *testing.B) {
	for i := 0; i < b.N; i++ {
		var c1 = "bytes 0-100/*"
		httpRequestParseContentRangeHeader(c1)
	}
}

func TestHTTPRequest_httpRequestNextId(t *testing.T) {
	teaconst.NodeId = 123
	teaconst.NodeIdString = "123"
	t.Log(httpRequestNextId())
	t.Log(httpRequestNextId())
	t.Log(httpRequestNextId())
	time.Sleep(1 * time.Second)
	t.Log(httpRequestNextId())
	t.Log(httpRequestNextId())
	time.Sleep(1 * time.Second)
	t.Log(httpRequestNextId())
}

func TestHTTPRequest_httpRequestNextId_Concurrent(t *testing.T) {
	var m = map[string]zero.Zero{}
	var locker = sync.Mutex{}

	var count = 4000
	var wg = &sync.WaitGroup{}
	wg.Add(count)

	var countDuplicated = 0
	for i := 0; i < count; i++ {
		go func() {
			defer wg.Done()

			var requestId = httpRequestNextId()

			locker.Lock()

			_, ok := m[requestId]
			if ok {
				t.Log("duplicated:", requestId)
				countDuplicated++
			}

			m[requestId] = zero.New()
			locker.Unlock()
		}()
	}
	wg.Wait()
	t.Log("ok", countDuplicated, "duplicated")

	var a = assert.NewAssertion(t)
	a.IsTrue(countDuplicated == 0)
}

func TestHTTPParseURL(t *testing.T) {
	for _, s := range []string{
		"",
		"null",
		"example.com",
		"https://example.com",
		"https://example.com/hello",
	} {
		host, err := httpParseHost(s)
		if err == nil {
			t.Log(s, "=>", host)
		} else {
			t.Log(s, "=>")
		}
	}
}

func BenchmarkHTTPRequest_httpRequestNextId(b *testing.B) {
	runtime.GOMAXPROCS(1)

	teaconst.NodeIdString = "123"

	for i := 0; i < b.N; i++ {
		_ = httpRequestNextId()
	}
}
