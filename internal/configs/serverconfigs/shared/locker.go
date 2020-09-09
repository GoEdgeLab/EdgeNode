package shared

import (
	"sync"
)

var Locker = new(FileLocker)

// global file modify locker
type FileLocker struct {
	locker sync.Mutex
}

// lock
func (this *FileLocker) Lock() {
	this.locker.Lock()
}

func (this *FileLocker) Unlock() {
	this.locker.Unlock()
}
