// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved. Official site: https://goedge.cn .

package readers_test

import (
	"bytes"
	"github.com/TeaOSLab/EdgeNode/internal/utils/readers"
	"io"
	"sync"
	"testing"
	"time"
)

type testReader struct {
	t *testing.T

	rawReader io.Reader
}

func (this *testReader) Read(p []byte) (n int, err error) {
	time.Sleep(1 * time.Second) // 延迟
	return this.rawReader.Read(p)
}

func (this *testReader) Close() error {
	this.t.Log("close")
	return nil
}

func TestNewConcurrentReader(t *testing.T) {
	var originBuffer = &bytes.Buffer{}
	originBuffer.Write([]byte("0123456789_hello_world"))
	var originLength = originBuffer.Len()
	var concurrentReader = readers.NewConcurrentReaderList(&testReader{
		t:         t,
		rawReader: originBuffer,
	})

	var threads = 32
	var wg = &sync.WaitGroup{}
	wg.Add(threads)

	var locker = &sync.Mutex{}
	var m = map[int][]byte{} // i => []byte

	for i := 0; i < threads; i++ {
		go func(i int) {
			defer wg.Done()

			var reader = concurrentReader.NewReader()

			var buf = make([]byte, 4)
			for {
				n, err := reader.Read(buf)
				if n > 0 {
					locker.Lock()
					m[i] = append(m[i], buf[:n]...)
					locker.Unlock()
					//t.Log(i, string(buf[:n]))
				}
				if err != nil {
					if err == io.EOF {
						break
					}
					t.Log("ERROR:", err)
				}
			}

			_ = reader.Close()
		}(i)
	}

	wg.Wait()

	for i, b := range m {
		if len(b) != originLength {
			t.Fatal("ERROR:", i, string(b))
		}
		t.Log(i, string(b))
	}
}
