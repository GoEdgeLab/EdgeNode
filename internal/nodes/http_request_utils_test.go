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

func TestHTTPRequest_httpRequestParseContentRange(t *testing.T) {
	a := assert.NewAssertion(t)
	{
		_, ok := httpRequestParseContentRange("")
		a.IsFalse(ok)
	}
	{
		_, ok := httpRequestParseContentRange("byte=")
		a.IsFalse(ok)
	}
	{
		_, ok := httpRequestParseContentRange("byte=")
		a.IsFalse(ok)
	}
	{
		set, ok := httpRequestParseContentRange("bytes=")
		a.IsTrue(ok)
		a.IsTrue(len(set) == 0)
	}
	{
		_, ok := httpRequestParseContentRange("bytes=60-50")
		a.IsFalse(ok)
	}
	{
		set, ok := httpRequestParseContentRange("bytes=0-50")
		a.IsTrue(ok)
		a.IsTrue(len(set) > 0)
		t.Log(set)
	}
	{
		set, ok := httpRequestParseContentRange("bytes=0-")
		a.IsTrue(ok)
		a.IsTrue(len(set) > 0)
		t.Log(set)
	}
	{
		set, ok := httpRequestParseContentRange("bytes=-50")
		a.IsTrue(ok)
		a.IsTrue(len(set) > 0)
		t.Log(set)
	}
	{
		set, ok := httpRequestParseContentRange("bytes=0-50, 60-100")
		a.IsTrue(ok)
		a.IsTrue(len(set) > 0)
		t.Log(set)
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

func BenchmarkHTTPRequest_httpRequestNextId(b *testing.B) {
	runtime.GOMAXPROCS(1)

	teaconst.NodeIdString = "123"

	for i := 0; i < b.N; i++ {
		_ = httpRequestNextId()
	}
}
