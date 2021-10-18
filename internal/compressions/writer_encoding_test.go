// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package compressions

import (
	"bytes"
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs"
	"testing"
)

func TestNewEncodingWriter(t *testing.T) {
	var buf = &bytes.Buffer{}

	subWriter, err := NewWriter(buf, serverconfigs.HTTPCompressionTypeGzip, 5)
	if err != nil {
		t.Fatal(err)
	}
	writer, err := NewEncodingWriter(ContentEncodingGzip, subWriter)
	if err != nil {
		t.Fatal(err)
	}

	gzipBuf := &bytes.Buffer{}
	gzipWriter, err := NewGzipWriter(gzipBuf, 5)
	if err != nil {
		t.Fatal(err)
	}
	_, err = gzipWriter.Write([]byte("Hello"))
	if err != nil {
		t.Fatal(err)
	}

	_, err = gzipWriter.Write([]byte("World"))
	if err != nil {
		t.Fatal(err)
	}
	_ = gzipWriter.Close()

	_, err = writer.Write(gzipBuf.Bytes())
	if err != nil {
		t.Fatal(err)
	}

	_ = writer.Close()

	t.Log(buf.String())
}
