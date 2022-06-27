// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package compressions_test

import (
	"bytes"
	"github.com/TeaOSLab/EdgeNode/internal/compressions"
	"strings"
	"testing"
)

func TestNewZSTDWriter(t *testing.T) {
	var buf = &bytes.Buffer{}
	writer, err := compressions.NewZSTDWriter(buf, 10)
	if err != nil {
		t.Fatal(err)
	}
	var originData = []byte(strings.Repeat("Hello", 1024))
	_, err = writer.Write(originData)
	if err != nil {
		t.Fatal(err)
	}
	err = writer.Close()
	if err != nil {
		t.Fatal(err)
	}
	t.Log("origin data:", len(originData), "result:", buf.Len())
}

func BenchmarkZSTDWriter_Write(b *testing.B) {
	var data = []byte(strings.Repeat("A", 1024))

	for i := 0; i < b.N; i++ {
		var buf = &bytes.Buffer{}
		writer, err := compressions.NewZSTDWriter(buf, 5)
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

func BenchmarkZSTDWriter_Write_Parallel(b *testing.B) {
	var data = []byte(strings.Repeat("A", 1024))

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			var buf = &bytes.Buffer{}
			writer, err := compressions.NewZSTDWriter(buf, 5)
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
