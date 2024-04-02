// Copyright 2024 GoEdge CDN goedge.cdn@gmail.com. All rights reserved. Official site: https://goedge.cn .

package metrics_test

import (
	"github.com/TeaOSLab/EdgeNode/internal/metrics"
	"github.com/iwind/TeaGo/assert"
	"testing"
)

func TestStat_EncodeValueKey(t *testing.T) {
	var a = assert.NewAssertion(t)

	var stat = &metrics.Stat{
		ServerId: 1,
		Keys:     []string{"${remoteAddr}"},
		Hash:     "123456",
		Value:    123,
		Time:     "20240101",
	}

	var valueKey = stat.EncodeValueKey(100)
	t.Log(valueKey)

	serverId, timeString, version, value, hash, err := metrics.DecodeValueKey(valueKey)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(serverId, timeString, value, version, hash)
	a.IsTrue(serverId == 1)
	a.IsTrue(timeString == "20240101")
	a.IsTrue(value == 123)
	a.IsTrue(version == 100)
	a.IsTrue(hash == "123456")
}

func TestStat_EncodeSumKey(t *testing.T) {
	var a = assert.NewAssertion(t)

	var stat = &metrics.Stat{
		ServerId: 1,
		Keys:     []string{"${remoteAddr}"},
		Hash:     "123456",
		Value:    123,
		Time:     "20240101",
	}
	var sumKey = stat.EncodeSumKey(100)
	t.Log(sumKey)

	serverId, timeString, version, err := metrics.DecodeSumKey(sumKey)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(serverId, timeString, version)
	a.IsTrue(serverId == 1)
	a.IsTrue(timeString == "20240101")
	a.IsTrue(version == 100)
}

func TestStat_EncodeSumValue(t *testing.T) {
	var a = assert.NewAssertion(t)

	var b = metrics.EncodeSumValue(123, 456)
	t.Log(b)

	count, sum := metrics.DecodeSumValue(b)
	a.IsTrue(count == 123)
	a.IsTrue(sum == 456)
}
