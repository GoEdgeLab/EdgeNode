package caches

import (
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs"
	"github.com/iwind/TeaGo/Tea"
	"os"
	"testing"
)

func TestFileReader(t *testing.T) {
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

	fp, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			t.Log("file '" + path + "' not exists")
			return
		}
		t.Fatal(err)
	}
	defer func() {
		_ = fp.Close()
	}()
	reader := NewFileReader(fp)
	err = reader.Init()
	if err != nil {
		t.Fatal(err)
	}

	t.Log(reader.Status())

	buf := make([]byte, 10)
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

func TestFileReader_ReadHeader(t *testing.T) {
	var path = "/Users/WorkSpace/EdgeProject/EdgeCache/p43/12/6b/126bbed90fc80f2bdfb19558948b0d49.cache"
	fp, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			t.Log("'" + path + "' not exists")
			return
		}
		t.Fatal(err)
	}
	defer func() {
		_ = fp.Close()
	}()
	var reader = NewFileReader(fp)
	err = reader.Init()
	if err != nil {
		if os.IsNotExist(err) {
			t.Log("file '" + path + "' not exists")
			return
		}

		t.Fatal(err)
	}
	var buf = make([]byte, 16*1024)
	err = reader.ReadHeader(buf, func(n int) (goNext bool, err error) {
		t.Log("header:", string(buf[:n]))
		return
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestFileReader_Range(t *testing.T) {
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

	/**writer, err := storage.Open("my-number", time.Now().Unix()+30*86400, 200, 6, 10)
	if err != nil {
		t.Fatal(err)
	}
	_, err = writer.Write([]byte("Header"))
	if err != nil {
		t.Fatal(err)
	}
	_, err = writer.Write([]byte("0123456789"))
	if err != nil {
		t.Fatal(err)
	}
	_ = writer.Close()**/

	_, path, _ := storage.keyPath("my-number")

	fp, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			t.Log("'" + path + "' not exists")
			return
		}
		t.Fatal(err)
	}
	defer func() {
		_ = fp.Close()
	}()
	reader := NewFileReader(fp)
	err = reader.Init()
	if err != nil {
		t.Fatal(err)
	}

	buf := make([]byte, 6)
	{
		err = reader.ReadBodyRange(buf, 0, 0, func(n int) (goNext bool, err error) {
			t.Log("[0, 0]", "body:", string(buf[:n]))
			return true, nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	{
		err = reader.ReadBodyRange(buf, 7, 7, func(n int) (goNext bool, err error) {
			t.Log("[7, 7]", "body:", string(buf[:n]))
			return true, nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	{
		err = reader.ReadBodyRange(buf, 0, 10, func(n int) (goNext bool, err error) {
			t.Log("[0, 10]", "body:", string(buf[:n]))
			return true, nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	{
		err = reader.ReadBodyRange(buf, 3, 5, func(n int) (goNext bool, err error) {
			t.Log("[3, 5]", "body:", string(buf[:n]))
			return true, nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	{
		err = reader.ReadBodyRange(buf, -1, -3, func(n int) (goNext bool, err error) {
			t.Log("[, -3]", "body:", string(buf[:n]))
			return true, nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	{
		err = reader.ReadBodyRange(buf, 3, -1, func(n int) (goNext bool, err error) {
			t.Log("[3, ]", "body:", string(buf[:n]))
			return true, nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}
