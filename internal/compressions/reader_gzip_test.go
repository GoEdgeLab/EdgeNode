// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package compressions

import (
	"bytes"
	"errors"
	"github.com/TeaOSLab/EdgeNode/internal/caches"
	"io"
	"os"
	"testing"
)

func TestGzipReader(t *testing.T) {
	fp, err := os.Open("/Users/WorkSpace/EdgeProject/EdgeCache/p43/36/7e/367e02720713fe05b66573a1d69b4f0a.cache")
	if err != nil {
		// not fatal
		t.Log(err)
		return
	}
	defer func() {
		_ = fp.Close()
	}()

	var buf = make([]byte, 32*1024)
	cacheReader := caches.NewFileReader(fp)
	err = cacheReader.Init()
	if err != nil {
		t.Fatal(err)
	}
	var headerBuf = []byte{}
	err = cacheReader.ReadHeader(buf, func(n int) (goNext bool, err error) {
		headerBuf = append(headerBuf, buf[:n]...)
		for {
			nIndex := bytes.Index(headerBuf, []byte{'\n'})
			if nIndex >= 0 {
				row := headerBuf[:nIndex]
				spaceIndex := bytes.Index(row, []byte{':'})
				if spaceIndex <= 0 {
					return false, errors.New("invalid header '" + string(row) + "'")
				}

				headerBuf = headerBuf[nIndex+1:]
			} else {
				break
			}
		}
		return true, nil
	})

	reader, err := NewGzipReader(cacheReader)
	if err != nil {
		t.Fatal(err)
	}

	for {
		n, err := reader.Read(buf)
		if err != nil {
			if err != io.EOF {
				t.Fatal(err)
			} else {
				break
			}
		}
		t.Log(string(buf[:n]))
		_ = n
	}
}
