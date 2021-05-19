package caches

import (
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs"
	"github.com/iwind/TeaGo/logs"
	"github.com/iwind/TeaGo/rands"
	"strconv"
	"testing"
	"time"
)

func TestMemoryStorage_OpenWriter(t *testing.T) {
	storage := NewMemoryStorage(&serverconfigs.HTTPCachePolicy{})

	writer, err := storage.OpenWriter("abc", time.Now().Unix()+60, 200)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = writer.WriteHeader([]byte("Header"))
	_, _ = writer.Write([]byte("Hello"))
	_, _ = writer.Write([]byte(", World"))
	t.Log(storage.valuesMap)

	{
		reader, err := storage.OpenReader("abc")
		if err != nil {
			if err == ErrNotFound {
				t.Log("not found: abc")
			} else {
				t.Fatal(err)
			}
		}
		buf := make([]byte, 1024)
		t.Log("status:", reader.Status())
		err = reader.ReadHeader(buf, func(n int) (goNext bool, err error) {
			t.Log("header:", string(buf[:n]))
			return true, nil
		})
		if err != nil {
			t.Fatal(err)
		}
		err = reader.ReadBody(buf, func(n int) (goNext bool, err error) {
			t.Log("body:", string(buf[:n]))
			return true, nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}

	{
		_, err := storage.OpenReader("abc 2")
		if err != nil {
			if err == ErrNotFound {
				t.Log("not found: abc2")
			} else {
				t.Fatal(err)
			}
		}
	}

	writer, err = storage.OpenWriter("abc", time.Now().Unix()+60, 200)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = writer.Write([]byte("Hello123"))
	{
		reader, err := storage.OpenReader("abc")
		if err != nil {
			if err == ErrNotFound {
				t.Log("not found: abc")
			} else {
				t.Fatal(err)
			}
		}
		buf := make([]byte, 1024)
		err = reader.ReadBody(buf, func(n int) (goNext bool, err error) {
			t.Log("abc:", string(buf[:n]))
			return true, nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestMemoryStorage_Delete(t *testing.T) {
	storage := NewMemoryStorage(&serverconfigs.HTTPCachePolicy{})
	{
		writer, err := storage.OpenWriter("abc", time.Now().Unix()+60, 200)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = writer.Write([]byte("Hello"))
		t.Log(len(storage.valuesMap))
	}
	{
		writer, err := storage.OpenWriter("abc1", time.Now().Unix()+60, 200)
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
		writer, err := storage.OpenWriter("abc", expiredAt, 200)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = writer.Write([]byte("Hello"))
		t.Log(len(storage.valuesMap))
		storage.AddToList(&Item{
			Key:       "abc",
			BodySize:  5,
			ExpiredAt: expiredAt,
		})
	}
	{
		writer, err := storage.OpenWriter("abc1", expiredAt, 200)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = writer.Write([]byte("Hello"))
		t.Log(len(storage.valuesMap))
		storage.AddToList(&Item{
			Key:       "abc1",
			BodySize:  5,
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
		writer, err := storage.OpenWriter("abc", expiredAt, 200)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = writer.Write([]byte("Hello"))
		storage.AddToList(&Item{
			Key:       "abc",
			BodySize:  5,
			ExpiredAt: expiredAt,
		})
	}
	{
		writer, err := storage.OpenWriter("abc1", expiredAt, 200)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = writer.Write([]byte("Hello"))
		storage.AddToList(&Item{
			Key:       "abc1",
			BodySize:  5,
			ExpiredAt: expiredAt,
		})
	}
	err := storage.CleanAll()
	if err != nil {
		t.Fatal(err)
	}
	total, _ := storage.list.Count()
	t.Log(total, len(storage.valuesMap))
}

func TestMemoryStorage_Purge(t *testing.T) {
	storage := NewMemoryStorage(&serverconfigs.HTTPCachePolicy{})
	expiredAt := time.Now().Unix() + 60
	{
		writer, err := storage.OpenWriter("abc", expiredAt, 200)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = writer.Write([]byte("Hello"))
		storage.AddToList(&Item{
			Key:       "abc",
			BodySize:  5,
			ExpiredAt: expiredAt,
		})
	}
	{
		writer, err := storage.OpenWriter("abc1", expiredAt, 200)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = writer.Write([]byte("Hello"))
		storage.AddToList(&Item{
			Key:       "abc1",
			BodySize:  5,
			ExpiredAt: expiredAt,
		})
	}
	err := storage.Purge([]string{"abc", "abc1"}, "")
	if err != nil {
		t.Fatal(err)
	}
	total, _ := storage.list.Count()
	t.Log(total, len(storage.valuesMap))
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
		writer, err := storage.OpenWriter(key, expiredAt, 200)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = writer.Write([]byte("Hello"))
		storage.AddToList(&Item{
			Key:       key,
			BodySize:  5,
			ExpiredAt: expiredAt,
		})
	}
	time.Sleep(70 * time.Second)
}
