// Copyright 2023 Liuxiangchao iwind.liu@gmail.com. All rights reserved. Official site: https://goedge.cn .

package nftables_test

import (
	"github.com/TeaOSLab/EdgeNode/internal/firewalls/nftables"
	"github.com/iwind/TeaGo/rands"
	"github.com/iwind/TeaGo/types"
	"net"
	"testing"
	"time"
)

func TestExpiration_Add(t *testing.T) {
	var expiration = nftables.NewExpiration()
	{
		expiration.Add([]byte{'a', 'b', 'c'}, time.Now())
		t.Log(expiration.Contains([]byte{'a', 'b', 'c'}))
	}
	{
		expiration.Add([]byte{'a', 'b', 'c'}, time.Now().Add(1*time.Second))
		t.Log(expiration.Contains([]byte{'a', 'b', 'c'}))
	}
	{
		expiration.Add([]byte{'a', 'b', 'c'}, time.Time{})
		t.Log(expiration.Contains([]byte{'a', 'b', 'c'}))
	}
	{
		expiration.Add([]byte{'a', 'b', 'c'}, time.Now().Add(-1*time.Second))
		t.Log(expiration.Contains([]byte{'a', 'b', 'c'}))
	}
	{
		expiration.Add([]byte{'a', 'b', 'c'}, time.Now().Add(-10*time.Second))
		t.Log(expiration.Contains([]byte{'a', 'b', 'c'}))
	}
	{
		expiration.Add([]byte{'a', 'b', 'c'}, time.Now().Add(1*time.Second))
		expiration.Remove([]byte{'a', 'b', 'c'})
		t.Log(expiration.Contains([]byte{'a', 'b', 'c'}))
	}
	{
		expiration.Add(net.ParseIP("10.254.0.75").To4(), time.Now())
		t.Log(expiration.Contains(net.ParseIP("10.254.0.75").To4()))
	}
}

func BenchmarkNewExpiration(b *testing.B) {
	var expiration = nftables.NewExpiration()
	for i := 0; i < 10_000; i++ {
		expiration.Add([]byte(types.String(types.String(rands.Int(0, 255))+"."+types.String(rands.Int(0, 255))+"."+types.String(rands.Int(0, 255))+"."+types.String(rands.Int(0, 255)))), time.Now().Add(3600*time.Second))
	}
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			expiration.Add([]byte(types.String(types.String(rands.Int(0, 255))+"."+types.String(rands.Int(0, 255))+"."+types.String(rands.Int(0, 255))+"."+types.String(rands.Int(0, 255)))), time.Now().Add(3600*time.Second))
		}
	})
}
