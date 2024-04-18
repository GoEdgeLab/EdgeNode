package ttlcache_test

import (
	"github.com/TeaOSLab/EdgeNode/internal/ttlcache"
	"github.com/TeaOSLab/EdgeNode/internal/utils/testutils"
	"github.com/TeaOSLab/EdgeNode/internal/zero"
	"github.com/cespare/xxhash/v2"
	"runtime"
	"strconv"
	"testing"
)

func TestHashCollision(t *testing.T) {
	var m = map[uint64]zero.Zero{}

	var count = 1_000
	if testutils.IsSingleTesting() {
		count = 100_000_000
	}

	for i := 0; i < count; i++ {
		var k = ttlcache.HashKeyString(strconv.Itoa(i))
		_, ok := m[k]
		if ok {
			t.Fatal("collision at", i)
		}
		m[k] = zero.New()
	}

	t.Log(len(m), "elements")
}

func BenchmarkHashKey_Bytes(b *testing.B) {
	runtime.GOMAXPROCS(1)
	for i := 0; i < b.N; i++ {
		ttlcache.HashKeyBytes([]byte("HELLO,WORLDHELLO,WORLDHELLO,WORLDHELLO,WORLDHELLO,WORLDHELLO,WORLD"))
	}
}

func BenchmarkHashKey_String(b *testing.B) {
	runtime.GOMAXPROCS(1)
	for i := 0; i < b.N; i++ {
		ttlcache.HashKeyString("HELLO,WORLDHELLO,WORLDHELLO,WORLDHELLO,WORLDHELLO,WORLDHELLO,WORLD")
	}
}

func BenchmarkHashKey_XXHash(b *testing.B) {
	runtime.GOMAXPROCS(1)
	for i := 0; i < b.N; i++ {
		xxhash.Sum64String("HELLO,WORLDHELLO,WORLDHELLO,WORLDHELLO,WORLDHELLO,WORLDHELLO,WORLD")
	}
}
