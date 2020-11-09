package iplibrary

import (
	"github.com/iwind/TeaGo/assert"
	"testing"
	"time"
)

func TestIPItem_Contains(t *testing.T) {
	a := assert.NewAssertion(t)

	{
		item := &IPItem{
			IPFrom:    IP2Long("192.168.1.100"),
			IPTo:      0,
			ExpiredAt: 0,
		}
		a.IsTrue(item.Contains(IP2Long("192.168.1.100")))
	}

	{
		item := &IPItem{
			IPFrom:    IP2Long("192.168.1.100"),
			IPTo:      0,
			ExpiredAt: time.Now().Unix() + 1,
		}
		a.IsTrue(item.Contains(IP2Long("192.168.1.100")))
	}

	{
		item := &IPItem{
			IPFrom:    IP2Long("192.168.1.100"),
			IPTo:      0,
			ExpiredAt: time.Now().Unix() - 1,
		}
		a.IsFalse(item.Contains(IP2Long("192.168.1.100")))
	}
	{
		item := &IPItem{
			IPFrom:    IP2Long("192.168.1.100"),
			IPTo:      0,
			ExpiredAt: 0,
		}
		a.IsFalse(item.Contains(IP2Long("192.168.1.101")))
	}

	{
		item := &IPItem{
			IPFrom:    IP2Long("192.168.1.1"),
			IPTo:      IP2Long("192.168.1.101"),
			ExpiredAt: 0,
		}
		a.IsTrue(item.Contains(IP2Long("192.168.1.100")))
	}

	{
		item := &IPItem{
			IPFrom:    IP2Long("192.168.1.1"),
			IPTo:      IP2Long("192.168.1.100"),
			ExpiredAt: 0,
		}
		a.IsTrue(item.Contains(IP2Long("192.168.1.100")))
	}

	{
		item := &IPItem{
			IPFrom:    IP2Long("192.168.1.1"),
			IPTo:      IP2Long("192.168.1.101"),
			ExpiredAt: 0,
		}
		a.IsTrue(item.Contains(IP2Long("192.168.1.1")))
	}
}
