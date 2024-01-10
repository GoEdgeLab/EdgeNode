// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package compressions_test

import (
	"bytes"
	"github.com/TeaOSLab/EdgeNode/internal/compressions"
	"io"
	"os"
	"testing"
)

func TestBrotliReader(t *testing.T) {
	for _, testString := range []string{"Hello", "World", "Ni", "Hao"} {
		t.Log("===", testString, "===")
		var buf = &bytes.Buffer{}
		writer, err := compressions.NewBrotliWriter(buf, 5)
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

		reader, err := compressions.NewBrotliReader(buf)
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

func BenchmarkBrotliReader(b *testing.B) {
	data, err := os.ReadFile("./reader_brotli.go")
	if err != nil {
		b.Fatal(err)
	}
	var buf = bytes.NewBuffer([]byte{})
	writer, err := compressions.NewBrotliWriter(buf, 5)
	if err != nil {
		b.Fatal(err)
	}
	_, err = writer.Write(data)
	err = writer.Close()
	if err != nil {
		b.Fatal(err)
	}
	var compressedData = buf.Bytes()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			reader, readerErr := compressions.NewBrotliReader(bytes.NewBuffer(compressedData))
			if readerErr != nil {
				b.Fatal(readerErr)
			}
			var readBuf = make([]byte, 1024)
			for {
				_, readErr := reader.Read(readBuf)
				if readErr != nil {
					if readErr != io.EOF {
						b.Fatal(readErr)
					}
					break
				}
			}
			closeErr := reader.Close()
			if closeErr != nil {
				b.Fatal(closeErr)
			}
		}
	})
}
