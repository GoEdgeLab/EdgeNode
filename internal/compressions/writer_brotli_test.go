// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package compressions_test

import (
	"bytes"
	"github.com/TeaOSLab/EdgeNode/internal/compressions"
	stringutil "github.com/iwind/TeaGo/utils/string"
	"strings"
	"testing"
	"time"
)

func TestBrotliWriter_LargeFile(t *testing.T) {
	var data = []byte{}
	for i := 0; i < 1024*1024; i++ {
		data = append(data, stringutil.Rand(32)...)
	}
	t.Log(len(data)/1024/1024, "M")

	var before = time.Now()
	defer func() {
		t.Log(time.Since(before).Seconds()*1000, "ms")
	}()

	var buf = &bytes.Buffer{}
	writer, err := compressions.NewBrotliWriter(buf, 5)
	if err != nil {
		t.Fatal(err)
	}

	var offset = 0
	var size = 4096
	for offset < len(data) {
		_, err = writer.Write(data[offset : offset+size])
		if err != nil {
			t.Fatal(err)
		}
		offset += size
	}

	err = writer.Close()
	if err != nil {
		t.Fatal(err)
	}
}

func BenchmarkBrotliWriter_Write(b *testing.B) {
	var data = []byte(strings.Repeat("A", 1024))

	for i := 0; i < b.N; i++ {
		var buf = &bytes.Buffer{}
		writer, err := compressions.NewBrotliWriter(buf, 5)
		if err != nil {
			b.Fatal(err)
		}

		for j := 0; j < 100; j++ {
			_, err = writer.Write(data)
			if err != nil {
				b.Fatal(err)
			}

			/**err = writer.Flush()
			if err != nil {
				b.Fatal(err)
			}**/
		}

		_ = writer.Close()
	}
}

func BenchmarkBrotliWriter_Write_Parallel(b *testing.B) {
	var data = []byte(strings.Repeat("A", 1024))

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			var buf = &bytes.Buffer{}
			writer, err := compressions.NewBrotliWriter(buf, 5)
			if err != nil {
				b.Fatal(err)
			}

			for j := 0; j < 100; j++ {
				_, err = writer.Write(data)
				if err != nil {
					b.Fatal(err)
				}

				/**err = writer.Flush()
				if err != nil {
					b.Fatal(err)
				}**/
			}

			_ = writer.Close()
		}
	})
}

func BenchmarkBrotliWriter_Write_Small(b *testing.B) {
	var data = []byte(strings.Repeat("A", 16))

	for i := 0; i < b.N; i++ {
		var buf = &bytes.Buffer{}
		writer, err := compressions.NewBrotliWriter(buf, 5)
		if err != nil {
			b.Fatal(err)
		}

		for j := 0; j < 100; j++ {
			_, err = writer.Write(data)
			if err != nil {
				b.Fatal(err)
			}

			/**err = writer.Flush()
			if err != nil {
				b.Fatal(err)
			}**/
		}

		_ = writer.Close()
	}
}

func BenchmarkBrotliWriter_Write_Large(b *testing.B) {
	var data = []byte(strings.Repeat("A", 4096))

	for i := 0; i < b.N; i++ {
		var buf = &bytes.Buffer{}
		writer, err := compressions.NewBrotliWriter(buf, 5)
		if err != nil {
			b.Fatal(err)
		}

		for j := 0; j < 100; j++ {
			_, err = writer.Write(data)
			if err != nil {
				b.Fatal(err)
			}

			/**err = writer.Flush()
			if err != nil {
				b.Fatal(err)
			}**/
		}

		_ = writer.Close()
	}
}
