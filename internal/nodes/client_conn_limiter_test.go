// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package nodes

import (
	"github.com/iwind/TeaGo/logs"
	"testing"
)

func TestClientConnLimiter_Add(t *testing.T) {
	var limiter = NewClientConnLimiter()
	{
		b := limiter.Add("127.0.0.1:1234", 1, "192.168.1.100", 10, 5)
		t.Log(b)
	}
	{
		b := limiter.Add("127.0.0.1:1235", 1, "192.168.1.100", 10, 5)
		t.Log(b)
	}
	{
		b := limiter.Add("127.0.0.1:1236", 1, "192.168.1.100", 10, 5)
		t.Log(b)
	}
	{
		b := limiter.Add("127.0.0.1:1237", 1, "192.168.1.101", 10, 5)
		t.Log(b)
	}
	{
		b := limiter.Add("127.0.0.1:1238", 1, "192.168.1.100", 5, 5)
		t.Log(b)
	}
	limiter.Remove("127.0.0.1:1238")
	limiter.Remove("127.0.0.1:1239")
	limiter.Remove("127.0.0.1:1237")
	logs.PrintAsJSON(limiter.remoteAddrMap, t)
	logs.PrintAsJSON(limiter.ipConns, t)
	logs.PrintAsJSON(limiter.serverConns, t)
}
