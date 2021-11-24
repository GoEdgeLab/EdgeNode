package utils

import (
	"github.com/iwind/TeaGo/assert"
	"testing"
)

func TestCleanPath(t *testing.T) {
	a := assert.NewAssertion(t)

	a.IsTrue(CleanPath("") == "/")
	a.IsTrue(CleanPath("/hello/world") == "/hello/world")
	a.IsTrue(CleanPath("\\hello\\world") == "/hello/world")
	a.IsTrue(CleanPath("/\\hello\\//world") == "/hello/world")
	a.IsTrue(CleanPath("hello/world") == "/hello/world")
	a.IsTrue(CleanPath("/hello////world") == "/hello/world")
}

func TestCleanPath_Args(t *testing.T) {
	a := assert.NewAssertion(t)
	a.IsTrue(CleanPath("/hello/world?base=///////") == "/hello/world?base=///////")
}

func BenchmarkCleanPath(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = CleanPath("/hello///world/very/long/very//long")
	}
}
