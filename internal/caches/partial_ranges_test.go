// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package caches_test

import (
	"github.com/TeaOSLab/EdgeNode/internal/caches"
	"github.com/iwind/TeaGo/assert"
	"github.com/iwind/TeaGo/logs"
	"testing"
)

func TestNewPartialRanges(t *testing.T) {
	var r = caches.NewPartialRanges()
	r.Add(1, 100)
	r.Add(50, 300)

	r.Add(30, 80)
	r.Add(30, 100)
	r.Add(30, 400)
	r.Add(1000, 10000)
	r.Add(200, 1000)
	r.Add(200, 10040)

	logs.PrintAsJSON(r.Ranges())
}

func TestNewPartialRanges1(t *testing.T) {
	var a = assert.NewAssertion(t)

	var r = caches.NewPartialRanges()
	r.Add(1, 100)
	r.Add(1, 101)
	r.Add(1, 102)
	r.Add(2, 103)
	r.Add(200, 300)
	r.Add(1, 1000)

	var rs = r.Ranges()
	logs.PrintAsJSON(rs, t)
	a.IsTrue(len(rs) == 1)
	if len(rs) == 1 {
		a.IsTrue(rs[0][0] == 1)
		a.IsTrue(rs[0][1] == 1000)
	}
}

func TestNewPartialRanges2(t *testing.T) {
	// low -> high
	var r = caches.NewPartialRanges()
	r.Add(1, 100)
	r.Add(1, 101)
	r.Add(1, 102)
	r.Add(2, 103)
	r.Add(200, 300)
	r.Add(301, 302)
	r.Add(303, 304)
	r.Add(250, 400)

	var rs = r.Ranges()
	logs.PrintAsJSON(rs, t)
}

func TestNewPartialRanges3(t *testing.T) {
	// high -> low
	var r = caches.NewPartialRanges()
	r.Add(301, 302)
	r.Add(303, 304)
	r.Add(200, 300)
	r.Add(250, 400)

	var rs = r.Ranges()
	logs.PrintAsJSON(rs, t)
}

func TestNewPartialRanges4(t *testing.T) {
	// nearby
	var r = caches.NewPartialRanges()
	r.Add(301, 302)
	r.Add(303, 304)
	r.Add(305, 306)

	r.Add(417, 417)
	r.Add(410, 415)
	r.Add(400, 409)

	var rs = r.Ranges()
	logs.PrintAsJSON(rs, t)
	t.Log(r.Contains(400, 416))
}

func TestNewPartialRanges5(t *testing.T) {
	var r = caches.NewPartialRanges()
	for j := 0; j < 1000; j++ {
		r.Add(int64(j), int64(j+100))
	}
	logs.PrintAsJSON(r.Ranges(), t)
}

func TestNewPartialRanges_AsJSON(t *testing.T) {
	var r = caches.NewPartialRanges()
	for j := 0; j < 1000; j++ {
		r.Add(int64(j), int64(j+100))
	}
	data, err := r.AsJSON()
	if err != nil {
		t.Fatal(err)
	}
	t.Log(string(data))

	r2, err := caches.NewPartialRangesFromJSON(data)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(r2.Ranges())
}

func BenchmarkNewPartialRanges(b *testing.B) {
	for i := 0; i < b.N; i++ {
		var r = caches.NewPartialRanges()
		for j := 0; j < 1000; j++ {
			r.Add(int64(j), int64(j+100))
		}
	}
}
