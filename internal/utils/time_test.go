package utils

import (
	"testing"
	"time"
)

func TestUnixTime(t *testing.T) {
	for i := 0; i < 5; i++ {
		t.Log(UnixTime(), "real:", time.Now().Unix())
		time.Sleep(1 * time.Second)
	}
}
