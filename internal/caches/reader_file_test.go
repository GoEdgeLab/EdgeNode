package caches

import (
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs"
	"github.com/iwind/TeaGo/Tea"
	"os"
	"testing"
)

func TestFileReader(t *testing.T) {
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

	fp, err := os.Open(path)
	if err != nil {
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
}

func TestFileReader_Range(t *testing.T) {
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

	_, path := storage.keyPath("my-number")

	fp, err := os.Open(path)
	if err != nil {
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
