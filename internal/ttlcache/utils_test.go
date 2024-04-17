package ttlcache

import (
	"github.com/cespare/xxhash/v2"
	"runtime"
	"testing"
)

func BenchmarkHashKey(b *testing.B) {
	runtime.GOMAXPROCS(1)
	for i := 0; i < b.N; i++ {
		HashKey([]byte("HELLO,WORLDHELLO,WORLDHELLO,WORLDHELLO,WORLDHELLO,WORLDHELLO,WORLD"))
	}
}

func BenchmarkHashKey2(b *testing.B) {
	runtime.GOMAXPROCS(1)
	for i := 0; i < b.N; i++ {
		xxhash.Sum64String("HELLO,WORLDHELLO,WORLDHELLO,WORLDHELLO,WORLDHELLO,WORLDHELLO,WORLD")
	}
}
