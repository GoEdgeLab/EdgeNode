package caches

import (
	"bytes"
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs"
	"github.com/TeaOSLab/EdgeNode/internal/utils"
	"github.com/iwind/TeaGo/Tea"
	_ "github.com/iwind/TeaGo/bootstrap"
	"github.com/iwind/TeaGo/logs"
	"runtime"
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
	t.Log(len(storage.list.m), "entries left")
}

func TestFileStorage_Open(t *testing.T) {
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

	writer, err := storage.Open("abc", time.Now().Unix()+3600)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(writer)

	_, err = writer.Write([]byte("Hello,World"))
	if err != nil {
		t.Fatal(err)
	}

	err = writer.Close()
	if err != nil {
		t.Fatal(err)
	}
}

func TestFileStorage_Write(t *testing.T) {
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
	reader := bytes.NewBuffer([]byte(`my_value
my_value2
my_value3
my_value4
my_value5
my_value6
my_value7
my_value8
my_value9
my_value10`))
	err = storage.Write("my-key", time.Now().Unix()+3600, reader)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("ok")
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
	t.Log(storage.Read("my-key", make([]byte, 64), func(data []byte, size int64, expiredAt int64, isEOF bool) {
		t.Log("[expiredAt]", "["+string(data)+"]")
	}))
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
	t.Log(storage.Read("my-key-10000", make([]byte, 64), func(data []byte, size int64, expiredAt int64, isEOF bool) {
		t.Log("[" + string(data) + "]")
	}))
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

	t.Log("before:", storage.list.m)

	err = storage.CleanAll()
	if err != nil {
		t.Fatal(err)
	}

	t.Log("after:", storage.list.m)
	t.Log("ok")
}

func TestFileStorage_Purge(t *testing.T) {
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

	_ = storage.Write("a", time.Now().Unix()+3600, bytes.NewReader([]byte("a1")))
	_ = storage.Write("b", time.Now().Unix()+3600, bytes.NewReader([]byte("b1")))
	_ = storage.Write("c", time.Now().Unix()+3600, bytes.NewReader([]byte("c1")))
	_ = storage.Write("d", time.Now().Unix()+3600, bytes.NewReader([]byte("d1")))

	before := time.Now()
	defer func() {
		t.Log(time.Since(before).Seconds()*1000, "ms")
	}()

	err = storage.Purge([]string{"a", "b1", "c"}, "")
	if err != nil {
		t.Fatal(err)
	}

	t.Log(storage.list.m)
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
	buf := make([]byte, 1024)
	for i := 0; i < b.N; i++ {
		_ = storage.Read("my-key", buf, func(data []byte, size int64, expiredAt int64, isEOF bool) {
		})
	}
}
