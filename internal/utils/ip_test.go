package utils

import (
	"github.com/iwind/TeaGo/assert"
	"testing"
)

func TestIP2Long(t *testing.T) {
	t.Log(IP2Long("0.0.0.0"))
	t.Log(IP2Long("1.0.0.0"))
	t.Log(IP2Long("0.0.0.0.0"))
	t.Log(IP2Long("2001:db8:0:1::101"))
	t.Log(IP2Long("2001:db8:0:1::102"))
	t.Log(IP2Long("::1"))
}

func TestIsLocalIP(t *testing.T) {
	var a = assert.NewAssertion(t)
	a.IsFalse(IsLocalIP("a"))
	a.IsFalse(IsLocalIP("1.2.3"))
	a.IsTrue(IsLocalIP("127.0.0.1"))
	a.IsTrue(IsLocalIP("192.168.0.1"))
	a.IsTrue(IsLocalIP("10.0.0.1"))
	a.IsTrue(IsLocalIP("172.16.0.1"))
	a.IsTrue(IsLocalIP("::1"))
	a.IsFalse(IsLocalIP("::1:2:3"))
	a.IsFalse(IsLocalIP("8.8.8.8"))
}
