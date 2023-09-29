package caches

import (
	"bytes"
	"errors"
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs"
	"github.com/TeaOSLab/EdgeNode/internal/utils"
	"github.com/TeaOSLab/EdgeNode/internal/utils/testutils"
	"github.com/iwind/TeaGo/Tea"
	_ "github.com/iwind/TeaGo/bootstrap"
	"github.com/iwind/TeaGo/logs"
	"io"
	"net/http"
	"runtime"
	"strconv"
	"sync"
	"testing"
	"time"
)

func TestFileStorage_Init(t *testing.T) {
	var storage = NewFileStorage(&serverconfigs.HTTPCachePolicy{
		Id:   1,
		IsOn: true,
		Options: map[string]interface{}{
			"dir": Tea.Root + "/caches",
		},
	})

	defer storage.Stop()

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
	t.Log(storage.list.(*FileList).Stat(func(hash string) bool {
		return true
	}))
}

func TestFileStorage_OpenWriter(t *testing.T) {
	var storage = NewFileStorage(&serverconfigs.HTTPCachePolicy{
		Id:   1,
		IsOn: true,
		Options: map[string]interface{}{
			"dir": Tea.Root + "/caches",
		},
	})

	defer storage.Stop()

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
	writer, err := storage.OpenWriter("my-key", time.Now().Unix()+86400, 200, -1, -1, -1, false)
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

func TestFileStorage_OpenWriter_Partial(t *testing.T) {
	var storage = NewFileStorage(&serverconfigs.HTTPCachePolicy{
		Id:   2,
		IsOn: true,
		Options: map[string]interface{}{
			"dir": Tea.Root + "/caches",
		},
	})

	defer storage.Stop()

	err := storage.Init()
	if err != nil {
		t.Fatal(err)
	}

	writer, err := storage.OpenWriter("my-key", time.Now().Unix()+86400, 200, -1, -1, -1, true)
	if err != nil {
		t.Fatal(err)
	}

	_, err = writer.WriteHeader([]byte("Content-Type:text/html; charset=utf-8"))
	if err != nil {
		t.Fatal(err)
	}

	err = writer.WriteAt(0, []byte("Hello, World"))
	if err != nil {
		t.Fatal(err)
	}

	err = writer.Close()
	if err != nil {
		t.Fatal(err)
	}
	t.Log(writer)
}

func TestFileStorage_OpenWriter_HTTP(t *testing.T) {
	var storage = NewFileStorage(&serverconfigs.HTTPCachePolicy{
		Id:   1,
		IsOn: true,
		Options: map[string]interface{}{
			"dir": Tea.Root + "/caches",
		},
	})

	defer storage.Stop()

	err := storage.Init()
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	defer func() {
		t.Log(time.Since(now).Seconds()*1000, "ms")
	}()

	writer, err := storage.OpenWriter("my-http-response", time.Now().Unix()+86400, 200, -1, -1, -1, false)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(writer)

	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header: http.Header{
			"Content-Type":  []string{"text/html; charset=utf-8"},
			"Last-Modified": []string{"Wed, 06 Jan 2021 10:03:29 GMT"},
			"Server":        []string{"CDN-Server"},
		},
		Body: io.NopCloser(bytes.NewBuffer([]byte("THIS IS HTTP BODY"))),
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
	var storage = NewFileStorage(&serverconfigs.HTTPCachePolicy{
		Id:   1,
		IsOn: true,
		Options: map[string]interface{}{
			"dir": Tea.Root + "/caches",
		},
	})

	defer storage.Stop()

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

			writer, err := storage.OpenWriter("abc"+strconv.Itoa(i), time.Now().Unix()+3600, 200, -1, -1, -1, false)
			if err != nil {
				if err != ErrFileIsWriting {
					t.Error(err)
					return
				}
				return
			}
			//t.Log(writer)

			_, err = writer.Write([]byte("Hello,World"))
			if err != nil {
				t.Error(err)
				return
			}

			// 故意造成慢速写入
			time.Sleep(1 * time.Second)

			err = writer.Close()
			if err != nil {
				t.Error(err)
				return
			}
		}(i)
	}

	wg.Wait()
}

func TestFileStorage_Concurrent_Open_SameFile(t *testing.T) {
	var storage = NewFileStorage(&serverconfigs.HTTPCachePolicy{
		Id:   1,
		IsOn: true,
		Options: map[string]interface{}{
			"dir": Tea.Root + "/caches",
		},
	})

	defer storage.Stop()

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

			writer, err := storage.OpenWriter("abc"+strconv.Itoa(0), time.Now().Unix()+3600, 200, -1, -1, -1, false)
			if err != nil {
				if err != ErrFileIsWriting {
					t.Error(err)
					return
				}
				return
			}
			//t.Log(writer)

			t.Log("writing")
			_, err = writer.Write([]byte("Hello,World"))
			if err != nil {
				t.Error(err)
				return
			}

			// 故意造成慢速写入
			time.Sleep(time.Duration(1) * time.Second)

			err = writer.Close()
			if err != nil {
				t.Error(err)
				return
			}
		}(i)
	}

	wg.Wait()
}

func TestFileStorage_Read(t *testing.T) {
	var storage = NewFileStorage(&serverconfigs.HTTPCachePolicy{
		Id:   1,
		IsOn: true,
		Options: map[string]interface{}{
			"dir": Tea.Root + "/caches",
		},
	})

	defer storage.Stop()

	err := storage.Init()
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	reader, err := storage.OpenReader("my-key", false, false)
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
	var storage = NewFileStorage(&serverconfigs.HTTPCachePolicy{
		Id:   1,
		IsOn: true,
		Options: map[string]interface{}{
			"dir": Tea.Root + "/caches",
		},
	})

	defer storage.Stop()

	err := storage.Init()
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	reader, err := storage.OpenReader("my-http-response", false, false)
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
	var storage = NewFileStorage(&serverconfigs.HTTPCachePolicy{
		Id:   1,
		IsOn: true,
		Options: map[string]interface{}{
			"dir": Tea.Root + "/caches",
		},
	})

	defer storage.Stop()

	err := storage.Init()
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	buf := make([]byte, 6)
	reader, err := storage.OpenReader("my-key-10000", false, false)
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
	var storage = NewFileStorage(&serverconfigs.HTTPCachePolicy{
		Id:   1,
		IsOn: true,
		Options: map[string]interface{}{
			"dir": Tea.Root + "/caches",
		},
	})

	defer storage.Stop()

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
	var storage = NewFileStorage(&serverconfigs.HTTPCachePolicy{
		Id:   1,
		IsOn: true,
		Options: map[string]interface{}{
			"dir": Tea.Root + "/caches",
		},
	})

	defer storage.Stop()

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
	var storage = NewFileStorage(&serverconfigs.HTTPCachePolicy{
		Id:   1,
		IsOn: true,
		Options: map[string]interface{}{
			"dir": Tea.Root + "/caches",
		},
	})

	defer storage.Stop()

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
	var storage = NewFileStorage(&serverconfigs.HTTPCachePolicy{
		Id:   1,
		IsOn: true,
		Options: map[string]interface{}{
			"dir": Tea.Root + "/caches",
		},
	})

	defer storage.Stop()

	err := storage.Init()
	if err != nil {
		t.Fatal(err)
	}
	storage.Stop()
}

func TestFileStorage_DecodeFile(t *testing.T) {
	var storage = NewFileStorage(&serverconfigs.HTTPCachePolicy{
		Id:   1,
		IsOn: true,
		Options: map[string]interface{}{
			"dir": Tea.Root + "/caches",
		},
	})

	defer storage.Stop()

	err := storage.Init()
	if err != nil {
		t.Fatal(err)
	}
	_, path, _ := storage.keyPath("my-key")
	t.Log(path)
}

func TestFileStorage_RemoveCacheFile(t *testing.T) {
	var storage = NewFileStorage(nil)

	defer storage.Stop()

	t.Log(storage.removeCacheFile("/Users/WorkSpace/EdgeProject/EdgeCache/p43/15/7e/157eba0dfc6dfb6fbbf20b1f9e584674.cache"))
}

func TestFileStorage_ScanGarbageCaches(t *testing.T) {
	if !testutils.IsSingleTesting() {
		return
	}

	var storage = NewFileStorage(&serverconfigs.HTTPCachePolicy{
		Id:      43,
		Options: map[string]any{"dir": "/Users/WorkSpace/EdgeProject/EdgeCache"},
	})
	err := storage.Init()
	if err != nil {
		t.Fatal(err)
	}

	err = storage.ScanGarbageCaches(func(path string) error {
		t.Log(path, PartialRangesFilePath(path))
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func BenchmarkFileStorage_Read(b *testing.B) {
	runtime.GOMAXPROCS(1)

	_ = utils.SetRLimit(1024 * 1024)

	var storage = NewFileStorage(&serverconfigs.HTTPCachePolicy{
		Id:   1,
		IsOn: true,
		Options: map[string]interface{}{
			"dir": Tea.Root + "/caches",
		},
	})

	defer storage.Stop()

	err := storage.Init()
	if err != nil {
		b.Fatal(err)
	}
	for i := 0; i < b.N; i++ {
		reader, err := storage.OpenReader("my-key", false, false)
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

func BenchmarkFileStorage_KeyPath(b *testing.B) {
	runtime.GOMAXPROCS(1)

	var storage = &FileStorage{
		options: &serverconfigs.HTTPFileCacheStorage{},
		policy:  &serverconfigs.HTTPCachePolicy{Id: 1},
	}

	for i := 0; i < b.N; i++ {
		_, _, _ = storage.keyPath(strconv.Itoa(i))
	}
}
