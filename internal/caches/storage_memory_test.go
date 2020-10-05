package caches

import (
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs"
	"github.com/iwind/TeaGo/logs"
	"github.com/iwind/TeaGo/rands"
	"strconv"
	"testing"
	"time"
)

func TestMemoryStorage_Open(t *testing.T) {
	storage := NewMemoryStorage(&serverconfigs.HTTPCachePolicy{})

	writer, err := storage.Open("abc", time.Now().Unix()+60)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = writer.Write([]byte("Hello"))
	_, _ = writer.Write([]byte(", World"))
	t.Log(storage.valuesMap)

	{
		err = storage.Read("abc", make([]byte, 8), func(data []byte, size int64, expiredAt int64, isEOF bool) {
			t.Log("read:", string(data))
		})
		if err != nil {
			if err == ErrNotFound {
				t.Log("not found: abc")
			} else {
				t.Fatal(err)
			}
		}
	}

	{
		err = storage.Read("abc 2", make([]byte, 8), func(data []byte, size int64, expiredAt int64, isEOF bool) {
			t.Log("read:", string(data))
		})
		if err != nil {
			if err == ErrNotFound {
				t.Log("not found: abc2")
			} else {
				t.Fatal(err)
			}
		}
	}

	writer, err = storage.Open("abc", time.Now().Unix()+60)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = writer.Write([]byte("Hello123"))
	{
		err = storage.Read("abc", make([]byte, 8), func(data []byte, size int64, expiredAt int64, isEOF bool) {
			t.Log("read:", string(data))
		})
		if err != nil {
			if err == ErrNotFound {
				t.Log("not found: abc")
			} else {
				t.Fatal(err)
			}
		}
	}
}

func TestMemoryStorage_Delete(t *testing.T) {
	storage := NewMemoryStorage(&serverconfigs.HTTPCachePolicy{})
	{
		writer, err := storage.Open("abc", time.Now().Unix()+60)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = writer.Write([]byte("Hello"))
		t.Log(len(storage.valuesMap))
	}
	{
		writer, err := storage.Open("abc1", time.Now().Unix()+60)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = writer.Write([]byte("Hello"))
		t.Log(len(storage.valuesMap))
	}
	_ = storage.Delete("abc1")
	t.Log(len(storage.valuesMap))
}

func TestMemoryStorage_Stat(t *testing.T) {
	storage := NewMemoryStorage(&serverconfigs.HTTPCachePolicy{})
	expiredAt := time.Now().Unix() + 60
	{
		writer, err := storage.Open("abc", expiredAt)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = writer.Write([]byte("Hello"))
		t.Log(len(storage.valuesMap))
		storage.AddToList(&Item{
			Key:       "abc",
			Size:      5,
			ExpiredAt: expiredAt,
		})
	}
	{
		writer, err := storage.Open("abc1", expiredAt)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = writer.Write([]byte("Hello"))
		t.Log(len(storage.valuesMap))
		storage.AddToList(&Item{
			Key:       "abc1",
			Size:      5,
			ExpiredAt: expiredAt,
		})
	}
	stat, err := storage.Stat()
	if err != nil {
		t.Fatal(err)
	}
	t.Log("===stat===")
	logs.PrintAsJSON(stat, t)
}

func TestMemoryStorage_CleanAll(t *testing.T) {
	storage := NewMemoryStorage(&serverconfigs.HTTPCachePolicy{})
	expiredAt := time.Now().Unix() + 60
	{
		writer, err := storage.Open("abc", expiredAt)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = writer.Write([]byte("Hello"))
		storage.AddToList(&Item{
			Key:       "abc",
			Size:      5,
			ExpiredAt: expiredAt,
		})
	}
	{
		writer, err := storage.Open("abc1", expiredAt)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = writer.Write([]byte("Hello"))
		storage.AddToList(&Item{
			Key:       "abc1",
			Size:      5,
			ExpiredAt: expiredAt,
		})
	}
	err := storage.CleanAll()
	if err != nil {
		t.Fatal(err)
	}
	t.Log(storage.list.Count(), len(storage.valuesMap))
}

func TestMemoryStorage_Purge(t *testing.T) {
	storage := NewMemoryStorage(&serverconfigs.HTTPCachePolicy{})
	expiredAt := time.Now().Unix() + 60
	{
		writer, err := storage.Open("abc", expiredAt)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = writer.Write([]byte("Hello"))
		storage.AddToList(&Item{
			Key:       "abc",
			Size:      5,
			ExpiredAt: expiredAt,
		})
	}
	{
		writer, err := storage.Open("abc1", expiredAt)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = writer.Write([]byte("Hello"))
		storage.AddToList(&Item{
			Key:       "abc1",
			Size:      5,
			ExpiredAt: expiredAt,
		})
	}
	err := storage.Purge([]string{"abc", "abc1"})
	if err != nil {
		t.Fatal(err)
	}
	t.Log(storage.list.Count(), len(storage.valuesMap))
}

func TestMemoryStorage_Expire(t *testing.T) {
	storage := NewMemoryStorage(&serverconfigs.HTTPCachePolicy{})
	storage.purgeDuration = 5 * time.Second
	err := storage.Init()
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 1000; i++ {
		expiredAt := time.Now().Unix() + int64(rands.Int(0, 60))
		key := "abc" + strconv.Itoa(i)
		writer, err := storage.Open(key, expiredAt)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = writer.Write([]byte("Hello"))
		storage.AddToList(&Item{
			Key:       key,
			Size:      5,
			ExpiredAt: expiredAt,
		})
	}
	time.Sleep(70 * time.Second)
}
