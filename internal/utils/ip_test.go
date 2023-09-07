package utils_test

import (
	"github.com/TeaOSLab/EdgeNode/internal/utils"
	"github.com/iwind/TeaGo/assert"
	"testing"
)

func TestIP2Long(t *testing.T) {
	t.Log(utils.IP2Long("0.0.0.0"))
	t.Log(utils.IP2Long("1.0.0.0"))
	t.Log(utils.IP2Long("0.0.0.0.0"))
	t.Log(utils.IP2Long("2001:db8:0:1::101"))
	t.Log(utils.IP2Long("2001:db8:0:1::102"))
	t.Log(utils.IP2Long("::1"))
}

func TestIsLocalIP(t *testing.T) {
	var a = assert.NewAssertion(t)
	a.IsFalse(utils.IsLocalIP("a"))
	a.IsFalse(utils.IsLocalIP("1.2.3"))
	a.IsTrue(utils.IsLocalIP("127.0.0.1"))
	a.IsTrue(utils.IsLocalIP("192.168.0.1"))
	a.IsTrue(utils.IsLocalIP("10.0.0.1"))
	a.IsTrue(utils.IsLocalIP("172.16.0.1"))
	a.IsTrue(utils.IsLocalIP("::1"))
	a.IsFalse(utils.IsLocalIP("::1:2:3"))
	a.IsFalse(utils.IsLocalIP("8.8.8.8"))
}

func TestIsIPv4(t *testing.T) {
	var a = assert.NewAssertion(t)
	a.IsTrue(utils.IsIPv4("192.168.1.1"))
	a.IsTrue(utils.IsIPv4("0.0.0.0"))
	a.IsFalse(utils.IsIPv4("192.168.1.256"))
	a.IsFalse(utils.IsIPv4("192.168.1"))
	a.IsFalse(utils.IsIPv4("::1"))
	a.IsFalse(utils.IsIPv4("2001:0db8:85a3:0000:0000:8a2e:0370:7334"))
	a.IsFalse(utils.IsIPv4("::ffff:192.168.0.1"))
}

func TestIsIPv6(t *testing.T) {
	var a = assert.NewAssertion(t)
	a.IsFalse(utils.IsIPv6("192.168.1.1"))
	a.IsFloat32(utils.IsIPv6("0.0.0.0"))
	a.IsFalse(utils.IsIPv6("192.168.1.256"))
	a.IsFalse(utils.IsIPv6("192.168.1"))
	a.IsTrue(utils.IsIPv6("::1"))
	a.IsTrue(utils.IsIPv6("2001:0db8:85a3:0000:0000:8a2e:0370:7334"))
	a.IsTrue(utils.IsIPv4("::ffff:192.168.0.1"))
	a.IsTrue(utils.IsIPv6("::ffff:192.168.0.1"))
}

func TestIsWildIP(t *testing.T) {
	var a = assert.NewAssertion(t)
	a.IsTrue(utils.IsWildIP("192.168.1.100"))
	a.IsTrue(utils.IsWildIP("::1"))
	a.IsTrue(utils.IsWildIP("2001:0db8:85a3:0000:0000:8a2e:0370:7334"))
	a.IsTrue(utils.IsWildIP("[2001:0db8:85a3:0000:0000:8a2e:0370:7334]"))
	a.IsFalse(utils.IsWildIP(""))
	a.IsFalse(utils.IsWildIP("[]"))
	a.IsFalse(utils.IsWildIP("[1]"))
	a.IsFalse(utils.IsWildIP("192.168.2.256"))
	a.IsFalse(utils.IsWildIP("192.168.2"))
}
