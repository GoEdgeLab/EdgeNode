package utils_test

import (
	"github.com/TeaOSLab/EdgeNode/internal/utils"
	"github.com/iwind/TeaGo/assert"
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
	a.IsFalse(utils.ContainsSameStrings([]string{"a"}, []string{"b"}))
	a.IsFalse(utils.ContainsSameStrings([]string{"a", "b"}, []string{"b"}))
	a.IsFalse(utils.ContainsSameStrings([]string{"a", "b"}, []string{"a", "b", "c"}))
	a.IsTrue(utils.ContainsSameStrings([]string{"a", "b"}, []string{"a", "b"}))
	a.IsTrue(utils.ContainsSameStrings([]string{"a", "b"}, []string{"b", "a"}))
}
