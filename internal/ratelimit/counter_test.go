// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package ratelimit

import (
	"testing"
	"time"
)

func TestCounter_ACK(t *testing.T) {
	var counter = NewCounter(10)

	go func() {
		for i := 0; i < 10; i++ {
			counter.Ack()
		}
		//counter.Release()
		t.Log("waiting", time.Now().Unix())
		counter.Ack()
		t.Log("done", time.Now().Unix())
	}()

	time.Sleep(1 * time.Second)
	counter.Close()
	time.Sleep(1 * time.Second)
}

func TestCounter_Release(t *testing.T) {
	var counter = NewCounter(10)

	for i := 0; i < 10; i++ {
		counter.Ack()
	}
	for i := 0; i < 10; i++ {
		counter.Release()
	}
	t.Log(len(counter.sem))
}
