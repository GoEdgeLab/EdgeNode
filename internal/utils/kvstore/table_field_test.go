// Copyright 2024 GoEdge CDN goedge.cdn@gmail.com. All rights reserved. Official site: https://goedge.cn .

package kvstore_test

import (
	"encoding/binary"
	"errors"
	"github.com/TeaOSLab/EdgeNode/internal/utils/fasttime"
	"github.com/TeaOSLab/EdgeNode/internal/utils/kvstore"
	"github.com/TeaOSLab/EdgeNode/internal/utils/testutils"
	"strconv"
	"testing"
	"time"
)

type testCachedItem struct {
	Hash       string `json:"1"` // as key
	URL        string `json:"2"`
	ExpiresAt  int64  `json:"3"`
	Tag        string `json:"tag"`
	HeaderSize int64  `json:"headerSize"`
	BodySize   int64  `json:"bodySize"`
	MetaSize   int    `json:"metaSize"`
	StaleAt    int64  `json:"staleAt"`
	CreatedAt  int64  `json:"createdAt"`
	Host       string `json:"host"`
	ServerId   int64  `json:"serverId"`
}

type testCacheItemEncoder[T interface{ *testCachedItem }] struct {
	kvstore.BaseObjectEncoder[T]
}

func (this *testCacheItemEncoder[T]) EncodeField(value T, fieldName string) ([]byte, error) {
	switch fieldName {
	case "expiresAt":
		var b = make([]byte, 4)
		binary.BigEndian.PutUint32(b, uint32(any(value).(*testCachedItem).ExpiresAt))
		return b, nil
	case "staleAt":
		var b = make([]byte, 4)
		binary.BigEndian.PutUint32(b, uint32(any(value).(*testCachedItem).StaleAt))
		return b, nil
	case "url":
		return []byte(any(value).(*testCachedItem).URL), nil
	}
	return nil, errors.New("EncodeField: invalid field name '" + fieldName + "'")
}

func TestTable_AddField(t *testing.T) {
	var table = testOpenStoreTable[*testCachedItem](t, "cache_items", &testCacheItemEncoder[*testCachedItem]{})

	err := table.AddFields("expiresAt")
	if err != nil {
		t.Fatal(err)
	}

	var before = time.Now()
	for _, item := range []*testCachedItem{
		{
			Hash:      "a1",
			URL:       "https://example.com/a1",
			ExpiresAt: 1710832067,
		},
		{
			Hash:      "a5",
			URL:       "https://example.com/a5",
			ExpiresAt: time.Now().Unix() + 7200,
		},
		{
			Hash:      "a4",
			URL:       "https://example.com/a4",
			ExpiresAt: time.Now().Unix() + 86400,
		},
		{
			Hash:      "a3",
			URL:       "https://example.com/a3",
			ExpiresAt: time.Now().Unix() + 1800,
		},
		{
			Hash:      "a2",
			URL:       "https://example.com/a2",
			ExpiresAt: time.Now().Unix() + 365*86400,
		},
	} {
		err = table.Set(item.Hash, item)
		if err != nil {
			t.Fatal(err)
		}
	}

	t.Log("set cost:", time.Since(before).Seconds()*1000, "ms")

	testInspectDB(t)
}

func TestTable_AddField_Many(t *testing.T) {
	if !testutils.IsSingleTesting() {
		return
	}

	//runtime.GOMAXPROCS(1)

	var table = testOpenStoreTable[*testCachedItem](t, "cache_items", &testCacheItemEncoder[*testCachedItem]{})

	{
		err := table.AddFields("expiresAt")
		if err != nil {
			t.Fatal(err)
		}
	}

	{
		err := table.AddFields("staleAt")
		if err != nil {
			t.Fatal(err)
		}
	}

	{
		err := table.AddFields("url")
		if err != nil {
			t.Fatal(err)
		}
	}

	var before = time.Now()
	const from = 0
	const count = 4_000_000

	defer func() {
		var costSeconds = time.Since(before).Seconds()
		t.Log("cost:", costSeconds*1000, "ms", "qps:", int64(float64(count)/costSeconds))
	}()

	for i := from; i < from+count; i++ {
		var item = &testCachedItem{
			Hash:      "a" + strconv.Itoa(i),
			URL:       "https://example.com/a" + strconv.Itoa(i),
			ExpiresAt: 1710832067 + int64(i),
			StaleAt:   fasttime.Now().Unix() + int64(i),
			CreatedAt: fasttime.Now().Unix(),
		}
		err := table.Set(item.Hash, item)
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestTable_AddField_Delete_Many(t *testing.T) {
	if !testutils.IsSingleTesting() {
		return
	}

	//runtime.GOMAXPROCS(1)

	var table = testOpenStoreTable[*testCachedItem](t, "cache_items", &testCacheItemEncoder[*testCachedItem]{})

	{
		err := table.AddFields("expiresAt")
		if err != nil {
			t.Fatal(err)
		}
	}

	{
		err := table.AddFields("staleAt")
		if err != nil {
			t.Fatal(err)
		}
	}

	{
		err := table.AddFields("url")
		if err != nil {
			t.Fatal(err)
		}
	}

	var before = time.Now()
	const from = 0
	const count = 1_000_000

	for i := from; i < from+count; i++ {
		var item = &testCachedItem{
			Hash: "a" + strconv.Itoa(i),
		}
		err := table.Delete(item.Hash)
		if err != nil {
			t.Fatal(err)
		}
	}

	var costSeconds = time.Since(before).Seconds()
	t.Log("cost:", costSeconds*1000, "ms", "qps:", int64(float64(count)/costSeconds))

	countLeft, err := table.Count()
	if err != nil {
		t.Fatal(err)
	}
	t.Log("left:", countLeft)
}

func TestTable_DropField(t *testing.T) {
	var table = testOpenStoreTable[*testCachedItem](t, "cache_items", &testCacheItemEncoder[*testCachedItem]{})

	var before = time.Now()
	defer func() {
		var costSeconds = time.Since(before).Seconds()
		t.Log("cost:", costSeconds*1000, "ms")
	}()

	err := table.DropField("expiresAt")
	if err != nil {
		t.Fatal(err)
	}
}

/**func TestTable_DeleteFieldValue(t *testing.T) {
	var table = testOpenStoreTable[*testCachedItem](t, "cache_items", &testCacheItemEncoder[*testCachedItem]{})
	err := table.AddField("expiresAt")
	if err != nil {
		t.Fatal(err)
	}

	var before = time.Now()
	defer func() {
		var costSeconds = time.Since(before).Seconds()
		t.Log("cost:", costSeconds*1000, "ms")
	}()

	err = table.Delete("a2")
	if err != nil {
		t.Fatal(err)
	}

	testInspectDB(t)
}
**/

func TestTable_Inspect(t *testing.T) {
	var table = testOpenStoreTable[*testCachedItem](t, "cache_items", &testCacheItemEncoder[*testCachedItem]{})

	err := table.AddFields("expiresAt")
	if err != nil {
		t.Fatal(err)
	}

	testInspectDB(t)
}
