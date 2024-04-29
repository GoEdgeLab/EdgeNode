// Copyright 2024 GoEdge CDN goedge.cdn@gmail.com. All rights reserved. Official site: https://goedge.cn .

package fsutils_test

import (
	fsutils "github.com/TeaOSLab/EdgeNode/internal/utils/fs"
	"github.com/iwind/TeaGo/assert"
	"testing"
)

func TestLimiter_Ack(t *testing.T) {
	var a = assert.NewAssertion(t)

	{
		var limiter = fsutils.NewLimiter(4)
		a.IsTrue(limiter.FreeThreads() == 4)
		limiter.Ack()
		a.IsTrue(limiter.FreeThreads() == 3)
		limiter.Ack()
		a.IsTrue(limiter.FreeThreads() == 2)
		limiter.Release()
		a.IsTrue(limiter.FreeThreads() == 3)
		limiter.Release()
		a.IsTrue(limiter.FreeThreads() == 4)
	}
}

func TestLimiter_TryAck(t *testing.T) {
	var a = assert.NewAssertion(t)

	{
		var limiter = fsutils.NewLimiter(4)
		var count = limiter.FreeThreads()
		a.IsTrue(count == 4)
		for i := 0; i < count; i++ {
			limiter.Ack()
		}
		a.IsTrue(limiter.FreeThreads() == 0)
		a.IsFalse(limiter.TryAck())
		a.IsTrue(limiter.FreeThreads() == 0)
	}

	{
		var limiter = fsutils.NewLimiter(4)
		var count = limiter.FreeThreads()
		a.IsTrue(count == 4)
		for i := 0; i < count-1; i++ {
			limiter.Ack()
		}
		a.IsTrue(limiter.FreeThreads() == 1)
		a.IsTrue(limiter.TryAck())
		a.IsTrue(limiter.FreeThreads() == 0)
	}
}
