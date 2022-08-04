// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package readers_test

import (
	"bytes"
	"fmt"
	"github.com/TeaOSLab/EdgeNode/internal/utils/readers"
	"io"
	"net/textproto"
	"testing"
)

func TestNewByteRangesReader(t *testing.T) {
	var boundary = "7143cd51d2ee12a1"
	var dashBoundary = "--" + boundary
	var b = bytes.NewReader([]byte(dashBoundary + "\r\nContent-Range: bytes 0-4/36\r\nContent-Type: text/plain\r\n\r\n01234\r\n" + dashBoundary + "\r\nContent-Range: bytes 5-9/36\r\nContent-Type: text/plain\r\n\r\n56789\r\n--" + boundary + "\r\nContent-Range: bytes 10-12/36\r\nContent-Type: text/plain\r\n\r\nabc\r\n" + dashBoundary + "--\r\n"))

	var reader = readers.NewByteRangesReaderCloser(io.NopCloser(b), boundary)
	var p = make([]byte, 16)
	for {
		n, err := reader.Read(p)
		if n > 0 {
			fmt.Print(string(p[:n]))
		}
		if err != nil {
			if err != io.EOF {
				t.Fatal(err)
			}
			break
		}
	}
}

func TestByteRangesReader_OnPartRead(t *testing.T) {
	var boundary = "7143cd51d2ee12a1"
	var dashBoundary = "--" + boundary
	var b = bytes.NewReader([]byte(dashBoundary + "\r\nContent-Range: bytes 0-4/36\r\nContent-Type: text/plain\r\n\r\n01234\r\n" + dashBoundary + "\r\nContent-Range: bytes 5-9/36\r\nContent-Type: text/plain\r\n\r\n56789\r\n--" + boundary + "\r\nContent-Range: bytes 10-12/36\r\nContent-Type: text/plain\r\n\r\nabc\r\n" + dashBoundary + "--\r\n"))

	var reader = readers.NewByteRangesReaderCloser(io.NopCloser(b), boundary)
	reader.OnPartRead(func(start int64, end int64, total int64, data []byte, header textproto.MIMEHeader) {
		t.Log(start, "-", end, "/", total, string(data))
	})
	var p = make([]byte, 3)
	for {
		_, err := reader.Read(p)
		if err != nil {
			break
		}
	}
}
