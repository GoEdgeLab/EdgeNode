package nodes

import (
	"github.com/iwind/TeaGo/assert"
	"testing"
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
