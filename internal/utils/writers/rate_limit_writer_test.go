// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package writers

import (
	"sync"
	"testing"
	"time"
)

func TestSleep(t *testing.T) {
	var count = 2000
	var wg = sync.WaitGroup{}
	wg.Add(count)
	var before = time.Now()
	for i := 0; i < count; i++ {
		go func() {
			defer wg.Done()
			time.Sleep(1 * time.Second)
		}()
	}
	wg.Wait()
	t.Log(time.Since(before).Seconds()*1000, "ms")
}

func TestTimeout(t *testing.T) {
	var count = 2000
	var wg = sync.WaitGroup{}
	wg.Add(count)
	var before = time.Now()
	for i := 0; i < count; i++ {
		go func() {
			defer wg.Done()

			var timeout = time.NewTimer(1 * time.Second)
			<-timeout.C
		}()
	}
	wg.Wait()
	t.Log(time.Since(before).Seconds()*1000, "ms")
}
