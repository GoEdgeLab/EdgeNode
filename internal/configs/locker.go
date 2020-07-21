package configs

import "sync"

var sharedLocker = &sync.RWMutex{}
