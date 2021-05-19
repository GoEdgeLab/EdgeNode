package caches

import (
	"bytes"
	"errors"
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs"
	"github.com/TeaOSLab/EdgeNode/internal/utils"
	"github.com/iwind/TeaGo/Tea"
	_ "github.com/iwind/TeaGo/bootstrap"
	"github.com/iwind/TeaGo/logs"
	"io/ioutil"
	"net/http"
	"runtime"
	"strconv"
	"sync"
	"testing"
	"time"
)

func TestFileStorage_Init(t *testing.T) {
	storage := NewFileStorage(&serverconfigs.HTTPCachePolicy{
		Id:   1,
		IsOn: true,
		Options: map[string]interface{}{
			"dir": Tea.Root + "/caches",
		},
	})

	err := storage.Init()
	if err != nil {
		t.Fatal(err)
	}
	//t.Log(storage.list.m)

	/**err = storage.Write("c", bytes.NewReader([]byte("i am c")), 4, "second")
	if err != nil {
		t.Fatal(err)
	}**/
	//logs.PrintAsJSON(storage.list.m, t)

	time.Sleep(2 * time.Second)
	storage.purgeLoop()
	t.Log(storage.list.(*FileList).total, "entries left")
}

func TestFileStorage_OpenWriter(t *testing.T) {
	storage := NewFileStorage(&serverconfigs.HTTPCachePolicy{
		Id:   1,
		IsOn: true,
		Options: map[string]interface{}{
			"dir": Tea.Root + "/caches",
		},
	})
	err := storage.Init()
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	defer func() {
		t.Log(time.Since(now).Seconds()*1000, "ms")
	}()

	header := []byte("Header")
	body := []byte("This is Body")
	writer, err := storage.OpenWriter("my-key", time.Now().Unix()+86400, 200)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(writer)

	_, err = writer.WriteHeader(header)
	if err != nil {
		t.Fatal(err)
	}

	_, err = writer.Write(body)
	if err != nil {
		t.Fatal(err)
	}

	err = writer.Close()
	if err != nil {
		t.Fatal(err)
	}

	t.Log("header:", writer.HeaderSize(), "body:", writer.BodySize())
	t.Log("ok")
}

func TestFileStorage_OpenWriter_HTTP(t *testing.T) {
	storage := NewFileStorage(&serverconfigs.HTTPCachePolicy{
		Id:   1,
		IsOn: true,
		Options: map[string]interface{}{
			"dir": Tea.Root + "/caches",
		},
	})
	err := storage.Init()
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	defer func() {
		t.Log(time.Since(now).Seconds()*1000, "ms")
	}()

	writer, err := storage.OpenWriter("my-http-response", time.Now().Unix()+86400, 200)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(writer)

	resp := &http.Response{
		StatusCode: 200,
		Header: http.Header{
			"Content-Type":  []string{"text/html; charset=utf-8"},
			"Last-Modified": []string{"Wed, 06 Jan 2021 10:03:29 GMT"},
			"Server":        []string{"CDN-Server"},
		},
		Body: ioutil.NopCloser(bytes.NewBuffer([]byte("THIS IS HTTP BODY"))),
	}

	for k, v := range resp.Header {
		for _, v1 := range v {
			_, err = writer.WriteHeader([]byte(k + ":" + v1 + "\n"))
			if err != nil {
				t.Fatal(err)
			}
		}
	}

	buf := make([]byte, 1024)
	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			_, err = writer.Write(buf[:n])
			if err != nil {
				t.Fatal(err)
			}
		}
		if err != nil {
			break
		}
	}

	err = writer.Close()
	if err != nil {
		t.Fatal(err)
	}

	t.Log("header:", writer.HeaderSize(), "body:", writer.BodySize())
	t.Log("ok")
}

func TestFileStorage_Concurrent_Open_DifferentFile(t *testing.T) {
	storage := NewFileStorage(&serverconfigs.HTTPCachePolicy{
		Id:   1,
		IsOn: true,
		Options: map[string]interface{}{
			"dir": Tea.Root + "/caches",
		},
	})
	err := storage.Init()
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	defer func() {
		t.Log(time.Since(now).Seconds()*1000, "ms")
	}()

	wg := sync.WaitGroup{}
	count := 100
	wg.Add(count)

	for i := 0; i < count; i++ {
		go func(i int) {
			defer wg.Done()

			writer, err := storage.OpenWriter("abc"+strconv.Itoa(i), time.Now().Unix()+3600, 200)
			if err != nil {
				if err != ErrFileIsWriting {
					t.Fatal(err)
				}
				return
			}
			//t.Log(writer)

			_, err = writer.Write([]byte("Hello,World"))
			if err != nil {
				t.Fatal(err)
			}

			// 故意造成慢速写入
			time.Sleep(1 * time.Second)

			err = writer.Close()
			if err != nil {
				t.Fatal(err)
			}
		}(i)
	}

	wg.Wait()
}

func TestFileStorage_Concurrent_Open_SameFile(t *testing.T) {
	storage := NewFileStorage(&serverconfigs.HTTPCachePolicy{
		Id:   1,
		IsOn: true,
		Options: map[string]interface{}{
			"dir": Tea.Root + "/caches",
		},
	})
	err := storage.Init()
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	defer func() {
		t.Log(time.Since(now).Seconds()*1000, "ms")
	}()

	wg := sync.WaitGroup{}
	count := 100
	wg.Add(count)

	for i := 0; i < count; i++ {
		go func(i int) {
			defer wg.Done()

			writer, err := storage.OpenWriter("abc"+strconv.Itoa(0), time.Now().Unix()+3600, 200)
			if err != nil {
				if err != ErrFileIsWriting {
					t.Fatal(err)
				}
				return
			}
			//t.Log(writer)

			t.Log("writing")
			_, err = writer.Write([]byte("Hello,World"))
			if err != nil {
				t.Fatal(err)
			}

			// 故意造成慢速写入
			time.Sleep(time.Duration(1) * time.Second)

			err = writer.Close()
			if err != nil {
				t.Fatal(err)
			}
		}(i)
	}

	wg.Wait()
}

func TestFileStorage_Read(t *testing.T) {
	storage := NewFileStorage(&serverconfigs.HTTPCachePolicy{
		Id:   1,
		IsOn: true,
		Options: map[string]interface{}{
			"dir": Tea.Root + "/caches",
		},
	})
	err := storage.Init()
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	reader, err := storage.OpenReader("my-key")
	if err != nil {
		t.Fatal(err)
	}
	buf := make([]byte, 6)
	t.Log(reader.Status())
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
	t.Log(time.Since(now).Seconds()*1000, "ms")
}

func TestFileStorage_Read_HTTP_Response(t *testing.T) {
	storage := NewFileStorage(&serverconfigs.HTTPCachePolicy{
		Id:   1,
		IsOn: true,
		Options: map[string]interface{}{
			"dir": Tea.Root + "/caches",
		},
	})
	err := storage.Init()
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	reader, err := storage.OpenReader("my-http-response")
	if err != nil {
		t.Fatal(err)
	}
	buf := make([]byte, 32)
	t.Log(reader.Status())

	headerBuf := []byte{}
	err = reader.ReadHeader(buf, func(n int) (goNext bool, err error) {
		headerBuf = append(headerBuf, buf...)
		for {
			nIndex := bytes.Index(headerBuf, []byte{'\n'})
			if nIndex >= 0 {
				row := headerBuf[:nIndex]
				spaceIndex := bytes.Index(row, []byte{':'})
				if spaceIndex <= 0 {
					return false, errors.New("invalid header")
				}

				t.Log("header row:", string(row[:spaceIndex]), string(row[spaceIndex+1:]))
				headerBuf = headerBuf[nIndex+1:]
			} else {
				break
			}
		}
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
	t.Log(time.Since(now).Seconds()*1000, "ms")
}

func TestFileStorage_Read_NotFound(t *testing.T) {
	storage := NewFileStorage(&serverconfigs.HTTPCachePolicy{
		Id:   1,
		IsOn: true,
		Options: map[string]interface{}{
			"dir": Tea.Root + "/caches",
		},
	})
	err := storage.Init()
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	buf := make([]byte, 6)
	reader, err := storage.OpenReader("my-key-10000")
	if err != nil {
		if err == ErrNotFound {
			t.Log("cache not fund")
			return
		}
		t.Fatal(err)
	}

	err = reader.ReadBody(buf, func(n int) (goNext bool, err error) {
		t.Log("body:", string(buf[:n]))
		return true, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Log(time.Since(now).Seconds()*1000, "ms")
}

func TestFileStorage_Delete(t *testing.T) {
	storage := NewFileStorage(&serverconfigs.HTTPCachePolicy{
		Id:   1,
		IsOn: true,
		Options: map[string]interface{}{
			"dir": Tea.Root + "/caches",
		},
	})
	err := storage.Init()
	if err != nil {
		t.Fatal(err)
	}
	err = storage.Delete("my-key")
	if err != nil {
		t.Fatal(err)
	}
	t.Log("ok")
}

func TestFileStorage_Stat(t *testing.T) {
	storage := NewFileStorage(&serverconfigs.HTTPCachePolicy{
		Id:   1,
		IsOn: true,
		Options: map[string]interface{}{
			"dir": Tea.Root + "/caches",
		},
	})
	err := storage.Init()
	if err != nil {
		t.Fatal(err)
	}

	before := time.Now()
	defer func() {
		t.Log(time.Since(before).Seconds()*1000, "ms")
	}()

	stat, err := storage.Stat()
	if err != nil {
		t.Fatal(err)
	}
	logs.PrintAsJSON(stat, t)
}

func TestFileStorage_CleanAll(t *testing.T) {
	storage := NewFileStorage(&serverconfigs.HTTPCachePolicy{
		Id:   1,
		IsOn: true,
		Options: map[string]interface{}{
			"dir": Tea.Root + "/caches",
		},
	})
	err := storage.Init()
	if err != nil {
		t.Fatal(err)
	}

	before := time.Now()
	defer func() {
		t.Log(time.Since(before).Seconds()*1000, "ms")
	}()

	c, _ := storage.list.Count()
	t.Log("before:", c)

	err = storage.CleanAll()
	if err != nil {
		t.Fatal(err)
	}

	c, _ = storage.list.Count()
	t.Log("after:", c)
	t.Log("ok")
}

func TestFileStorage_Stop(t *testing.T) {
	storage := NewFileStorage(&serverconfigs.HTTPCachePolicy{
		Id:   1,
		IsOn: true,
		Options: map[string]interface{}{
			"dir": Tea.Root + "/caches",
		},
	})
	err := storage.Init()
	if err != nil {
		t.Fatal(err)
	}
	storage.Stop()
}

func TestFileStorage_DecodeFile(t *testing.T) {
	storage := NewFileStorage(&serverconfigs.HTTPCachePolicy{
		Id:   1,
		IsOn: true,
		Options: map[string]interface{}{
			"dir": Tea.Root + "/caches",
		},
	})
	err := storage.Init()
	if err != nil {
		t.Fatal(err)
	}
	_, path := storage.keyPath("my-key")
	item, err := storage.decodeFile(path)
	if err != nil {
		t.Fatal(err)
	}
	logs.PrintAsJSON(item, t)
}

func BenchmarkFileStorage_Read(b *testing.B) {
	runtime.GOMAXPROCS(1)

	_ = utils.SetRLimit(1024 * 1024)

	storage := NewFileStorage(&serverconfigs.HTTPCachePolicy{
		Id:   1,
		IsOn: true,
		Options: map[string]interface{}{
			"dir": Tea.Root + "/caches",
		},
	})
	err := storage.Init()
	if err != nil {
		b.Fatal(err)
	}
	for i := 0; i < b.N; i++ {
		reader, err := storage.OpenReader("my-key")
		if err != nil {
			b.Fatal(err)
		}
		buf := make([]byte, 1024)
		_ = reader.ReadBody(buf, func(n int) (goNext bool, err error) {
			return true, nil
		})
		_ = reader.Close()
	}
}
