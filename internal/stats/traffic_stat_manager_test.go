package stats

import (
	"github.com/iwind/TeaGo/rands"
	"github.com/iwind/TeaGo/types"
	"math/rand"
	"runtime"
	"testing"
)

func TestTrafficStatManager_Add(t *testing.T) {
	manager := NewTrafficStatManager()
	for i := 0; i < 100; i++ {
		manager.Add(1, 1, "goedge.cn", 1, 0, 0, 0, 0, 0, false, 0)
	}
	t.Log(manager.itemMap)
}

func TestTrafficStatManager_Upload(t *testing.T) {
	manager := NewTrafficStatManager()
	for i := 0; i < 100; i++ {
		manager.Add(1, 1, "goedge.cn"+types.String(rands.Int(0, 10)), 1, 0, 1, 0, 0, 0, false, 0)
	}
	err := manager.Upload()
	if err != nil {
		t.Fatal(err)
	}
	t.Log("ok")
}

func BenchmarkTrafficStatManager_Add(b *testing.B) {
	runtime.GOMAXPROCS(1)

	var manager = NewTrafficStatManager()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			manager.Add(1, 1, "goedge.cn"+types.String(rand.Int63()%10), 1024, 1, 0, 0, 0, 0, false, 0)
		}
	})
}
