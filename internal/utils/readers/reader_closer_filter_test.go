// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package readers_test

import (
	"bytes"
	"errors"
	"github.com/TeaOSLab/EdgeNode/internal/utils/readers"
	"testing"
)

func TestNewFilterReader(t *testing.T) {
	var reader = readers.NewFilterReaderCloser(bytes.NewBufferString("0123456789"))
	reader.Add(func(p []byte, err error) error {
		t.Log("filter1:", string(p), err)
		return nil
	})
	reader.Add(func(p []byte, err error) error {
		t.Log("filter2:", string(p), err)
		if string(p) == "345" {
			return errors.New("end")
		}
		return nil
	})
	reader.Add(func(p []byte, err error) error {
		t.Log("filter3:", string(p), err)
		return nil
	})

	var buf = make([]byte, 3)
	for {
		n, err := reader.Read(buf)
		if n > 0 {
			t.Log(string(buf[:n]))
		}
		if err != nil {
			t.Log(err)
			break
		}
	}
}
