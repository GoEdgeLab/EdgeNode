package stats

import (
	"runtime"
	"testing"
)

func TestTrafficStatManager_Add(t *testing.T) {
	manager := NewTrafficStatManager()
	for i := 0; i < 100; i++ {
		manager.Add(1, 10, 1, 0)
	}
	t.Log(manager.itemMap)
}

func TestTrafficStatManager_Upload(t *testing.T) {
	manager := NewTrafficStatManager()
	for i := 0; i < 100; i++ {
		manager.Add(1, 10, 1, 0)
	}
	err := manager.Upload()
	if err != nil {
		t.Fatal(err)
	}
	t.Log("ok")
}

func BenchmarkTrafficStatManager_Add(b *testing.B) {
	runtime.GOMAXPROCS(1)

	manager := NewTrafficStatManager()
	for i := 0; i < b.N; i++ {
		manager.Add(1, 1024, 1, 0)
	}
}
