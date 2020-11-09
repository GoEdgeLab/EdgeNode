package iplibrary

import (
	"runtime"
	"testing"
)

func TestIP2Long(t *testing.T) {
	t.Log(IP2Long("192.168.1.100"))
	t.Log(IP2Long("192.168.1.101"))
	t.Log(IP2Long("202.106.0.20"))
	t.Log(IP2Long("192.168.1")) // wrong ip, should return 0
}

func BenchmarkIP2Long(b *testing.B) {
	runtime.GOMAXPROCS(1)

	for i := 0; i < b.N; i++ {
		_ = IP2Long("192.168.1.100")
	}
}
