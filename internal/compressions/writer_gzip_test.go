// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package compressions_test

import (
	"bytes"
	"github.com/TeaOSLab/EdgeNode/internal/compressions"
	"testing"
)

func BenchmarkGzipWriter_Write(b *testing.B) {
	var data = make([]byte, 1024)
	for i := 0; i < 1024; i++ {
		data[i] = 'A' + byte(i%26)
	}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var buf = &bytes.Buffer{}
		writer, err := compressions.NewGzipWriter(buf, 5)
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

func BenchmarkGzipWriter_Write_Parallel(b *testing.B) {
	var data = make([]byte, 1024)
	for i := 0; i < 1024; i++ {
		data[i] = 'A' + byte(i%26)
	}

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			var buf = &bytes.Buffer{}
			writer, err := compressions.NewGzipWriter(buf, 5)
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
