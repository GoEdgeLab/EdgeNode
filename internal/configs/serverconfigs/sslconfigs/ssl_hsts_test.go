package sslconfigs

import (
	"github.com/iwind/TeaGo/assert"
	"testing"
)

func TestHSTSConfig(t *testing.T) {
	h := &HSTSConfig{}
	h.Init()
	t.Log(h.HeaderValue())

	h.IncludeSubDomains = true
	h.Init()
	t.Log(h.HeaderValue())

	h.Preload = true
	h.Init()
	t.Log(h.HeaderValue())

	h.IncludeSubDomains = false
	h.Init()
	t.Log(h.HeaderValue())

	h.MaxAge = 86400
	h.Init()
	t.Log(h.HeaderValue())

	a := assert.NewAssertion(t)
	a.IsTrue(h.Match("abc.com"))

	h.Domains = []string{"abc.com"}
	h.Init()
	a.IsTrue(h.Match("abc.com"))

	h.Domains = []string{"1.abc.com"}
	h.Init()
	a.IsFalse(h.Match("abc.com"))
}
