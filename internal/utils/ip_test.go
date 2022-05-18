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

func TestIsIPv4(t *testing.T) {
	var a = assert.NewAssertion(t)
	a.IsTrue(IsIPv4("192.168.1.1"))
	a.IsTrue(IsIPv4("0.0.0.0"))
	a.IsFalse(IsIPv4("192.168.1.256"))
	a.IsFalse(IsIPv4("192.168.1"))
	a.IsFalse(IsIPv4("::1"))
	a.IsFalse(IsIPv4("2001:0db8:85a3:0000:0000:8a2e:0370:7334"))
	a.IsFalse(IsIPv4("::ffff:192.168.0.1"))
}

func TestIsIPv6(t *testing.T) {
	var a = assert.NewAssertion(t)
	a.IsFalse(IsIPv6("192.168.1.1"))
	a.IsFloat32(IsIPv6("0.0.0.0"))
	a.IsFalse(IsIPv6("192.168.1.256"))
	a.IsFalse(IsIPv6("192.168.1"))
	a.IsTrue(IsIPv6("::1"))
	a.IsTrue(IsIPv6("2001:0db8:85a3:0000:0000:8a2e:0370:7334"))
	a.IsTrue(IsIPv4("::ffff:192.168.0.1"))
	a.IsTrue(IsIPv6("::ffff:192.168.0.1"))
}
