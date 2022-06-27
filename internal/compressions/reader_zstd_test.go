// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package compressions_test

import (
	"bytes"
	"github.com/TeaOSLab/EdgeNode/internal/compressions"
	"github.com/iwind/TeaGo/rands"
	"github.com/iwind/TeaGo/types"
	"io"
	"strings"
	"testing"
)

func TestZSTDReader(t *testing.T) {
	for _, testString := range []string{"Hello", "World", "Ni", "Hao"} {
		t.Log("===", testString, "===")
		var buf = &bytes.Buffer{}
		writer, err := compressions.NewZSTDWriter(buf, 5)
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

		reader, err := compressions.NewZSTDReader(buf)
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

func BenchmarkZSTDReader(b *testing.B) {
	var randomData = func() []byte {
		var b = strings.Builder{}
		for i := 0; i < 1024; i++ {
			b.WriteString(types.String(rands.Int64() % 10))
		}
		return []byte(b.String())
	}

	var buf = &bytes.Buffer{}
	writer, err := compressions.NewZSTDWriter(buf, 5)
	if err != nil {
		b.Fatal(err)
	}
	_, err = writer.Write(randomData())
	if err != nil {
		b.Fatal(err)
	}
	err = writer.Close()
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var newBytes = make([]byte, buf.Len())
		copy(newBytes, buf.Bytes())
		reader, err := compressions.NewZSTDReader(bytes.NewReader(newBytes))
		if err != nil {
			b.Fatal(err)
		}
		var data = make([]byte, 4096)
		for {
			n, err := reader.Read(data)
			if n > 0 {
				_ = data[:n]
			}
			if err != nil {
				if err == io.EOF {
					break
				}
				b.Fatal(err)
			}
		}
		err = reader.Close()
		if err != nil {
			b.Fatal(err)
		}
	}
}
