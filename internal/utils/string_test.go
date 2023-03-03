package utils_test

import (
	"github.com/TeaOSLab/EdgeNode/internal/utils"
	"github.com/iwind/TeaGo/assert"
	"github.com/iwind/TeaGo/types"
	"strings"
	"testing"
)

func TestBytesToString(t *testing.T) {
	t.Log(utils.UnsafeBytesToString([]byte("Hello,World")))
}

func TestStringToBytes(t *testing.T) {
	t.Log(string(utils.UnsafeStringToBytes("Hello,World")))
}

func BenchmarkBytesToString(b *testing.B) {
	var data = []byte("Hello,World")
	for i := 0; i < b.N; i++ {
		_ = utils.UnsafeBytesToString(data)
	}
}

func BenchmarkBytesToString2(b *testing.B) {
	var data = []byte("Hello,World")
	for i := 0; i < b.N; i++ {
		_ = string(data)
	}
}

func BenchmarkStringToBytes(b *testing.B) {
	var s = strings.Repeat("Hello,World", 1024)
	for i := 0; i < b.N; i++ {
		_ = utils.UnsafeStringToBytes(s)
	}
}

func BenchmarkStringToBytes2(b *testing.B) {
	var s = strings.Repeat("Hello,World", 1024)
	for i := 0; i < b.N; i++ {
		_ = []byte(s)
	}
}

func TestFormatAddress(t *testing.T) {
	t.Log(utils.FormatAddress("127.0.0.1:1234"))
	t.Log(utils.FormatAddress("127.0.0.1 : 1234"))
	t.Log(utils.FormatAddress("127.0.0.1：1234"))
}

func TestFormatAddressList(t *testing.T) {
	t.Log(utils.FormatAddressList([]string{
		"127.0.0.1:1234",
		"127.0.0.1 : 1234",
		"127.0.0.1：1234",
	}))
}

func TestContainsSameStrings(t *testing.T) {
	var a = assert.NewAssertion(t)
	a.IsFalse(utils.EqualStrings([]string{"a"}, []string{"b"}))
	a.IsFalse(utils.EqualStrings([]string{"a", "b"}, []string{"b"}))
	a.IsFalse(utils.EqualStrings([]string{"a", "b"}, []string{"a", "b", "c"}))
	a.IsTrue(utils.EqualStrings([]string{"a", "b"}, []string{"a", "b"}))
	a.IsTrue(utils.EqualStrings([]string{"a", "b"}, []string{"b", "a"}))
}

func TestToValidUTF8string(t *testing.T) {
	for _, s := range []string{
		"https://goedge.cn/",
		"提升mysql数据表写入速度",
		"😆",
		string([]byte{'a', 'b', 130, 131, 132, 133, 134, 'c'}),
	} {
		var u = utils.ToValidUTF8string(s)
		t.Log(s, "["+types.String(len(s))+"]", "=>", u, "["+types.String(len(u))+"]")
	}
}
