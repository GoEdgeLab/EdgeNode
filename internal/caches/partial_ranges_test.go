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

	logs.PrintAsJSON(r.Ranges, t)
	t.Log("max:", r.Max())
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

	var rs = r.Ranges
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

	var rs = r.Ranges
	logs.PrintAsJSON(rs, t)
}

func TestNewPartialRanges3(t *testing.T) {
	// high -> low
	var r = caches.NewPartialRanges()
	r.Add(301, 302)
	r.Add(303, 304)
	r.Add(200, 300)
	r.Add(250, 400)

	var rs = r.Ranges
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

	var rs = r.Ranges
	logs.PrintAsJSON(rs, t)
	t.Log(r.Contains(400, 416))
}

func TestNewPartialRanges5(t *testing.T) {
	var r = caches.NewPartialRanges()
	for j := 0; j < 1000; j++ {
		r.Add(int64(j), int64(j+100))
	}
	logs.PrintAsJSON(r.Ranges, t)
}

func TestNewPartialRanges_Nearest(t *testing.T) {
	{
		// nearby
		var r = caches.NewPartialRanges()
		r.Add(301, 400)
		r.Add(401, 500)
		r.Add(501, 600)

		t.Log(r.Nearest(100, 200))
		t.Log(r.Nearest(300, 350))
		t.Log(r.Nearest(302, 350))
	}

	{
		// nearby
		var r = caches.NewPartialRanges()
		r.Add(301, 400)
		r.Add(450, 500)
		r.Add(550, 600)

		t.Log(r.Nearest(100, 200))
		t.Log(r.Nearest(300, 350))
		t.Log(r.Nearest(302, 350))
		t.Log(r.Nearest(302, 440))
		t.Log(r.Nearest(302, 1000))
	}
}

func TestNewPartialRanges_Large_Range(t *testing.T) {
	var a = assert.NewAssertion(t)

	var largeSize int64 = 10000000000000
	t.Log(largeSize/1024/1024/1024, "G")

	var r = caches.NewPartialRanges()
	r.Add(1, largeSize)
	jsonData, err := r.AsJSON()
	if err != nil {
		t.Fatal(err)
	}
	t.Log(string(jsonData))

	r2, err := caches.NewPartialRangesFromJSON(jsonData)
	if err != nil {
		t.Fatal(err)
	}

	a.IsTrue(largeSize == r2.Ranges[0][1])
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
	t.Log(r2.Ranges)
}

func BenchmarkNewPartialRanges(b *testing.B) {
	for i := 0; i < b.N; i++ {
		var r = caches.NewPartialRanges()
		for j := 0; j < 1000; j++ {
			r.Add(int64(j), int64(j+100))
		}
	}
}
