// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package compressions_test

import (
	"bytes"
	"github.com/TeaOSLab/EdgeNode/internal/compressions"
	"io"
	"testing"
)

func TestDeflateReader(t *testing.T) {
	for _, testString := range []string{"Hello", "World", "Ni", "Hao"} {
		t.Log("===", testString, "===")
		var buf = &bytes.Buffer{}
		writer, err := compressions.NewDeflateWriter(buf, 5)
		if err != nil {
			t.Fatal(err)
		}
		_, err = writer.Write([]byte(testString))
		if err != nil {
			t.Fatal(err)
		}
		err = writer.Close()
		if err != nil {
			t.Fatal(err)
		}

		reader, err := compressions.NewDeflateReader(buf)
		if err != nil {
			t.Fatal(err)
		}
		var data = make([]byte, 4096)
		for {
			n, err := reader.Read(data)
			if n > 0 {
				t.Log(string(data[:n]))
			}
			if err != nil {
				if err == io.EOF {
					break
				}
				t.Fatal(err)
			}
		}
		err = reader.Close()
		if err != nil {
			t.Fatal(err)
		}
	}
}
