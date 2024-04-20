// Copyright 2024 GoEdge CDN goedge.cdn@gmail.com. All rights reserved. Official site: https://goedge.cn .

package caches_test

import (
	"github.com/TeaOSLab/EdgeNode/internal/caches"
	"github.com/iwind/TeaGo/assert"
	"testing"
)

func TestNewPartialRangesQueue(t *testing.T) {
	var a = assert.NewAssertion(t)

	var queue = caches.NewPartialRangesQueue()
	queue.Put("a", []byte{1, 2, 3})
	t.Log("add 'a':", queue.Len())
	t.Log(queue.Get("a"))
	a.IsTrue(queue.Len() == 1)

	queue.Put("a", nil)
	t.Log("add 'a':", queue.Len())
	a.IsTrue(queue.Len() == 1)

	queue.Put("b", nil)
	t.Log("add 'b':", queue.Len())
	a.IsTrue(queue.Len() == 2)

	queue.Delete("a")
	t.Log("delete 'a':", queue.Len())
	a.IsTrue(queue.Len() == 1)
}
