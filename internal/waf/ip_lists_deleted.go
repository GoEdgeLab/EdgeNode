// Copyright 2023 GoEdge CDN goedge.cdn@gmail.com. All rights reserved. Official site: https://goedge.cn .

package waf

import (
	"github.com/TeaOSLab/EdgeNode/internal/zero"
	"sync"
)

var deletedIPListIdMap = map[int64]zero.Zero{} // listId => Zero
var deletedIPListLocker = sync.RWMutex{}

// AddDeletedIPList add deleted ip list
func AddDeletedIPList(ipListId int64) {
	if ipListId <= 0 {
		return
	}

	deletedIPListLocker.Lock()
	deletedIPListIdMap[ipListId] = zero.Zero{}
	deletedIPListLocker.Unlock()
}

// ExistDeletedIPList check if ip list has been deleted
func ExistDeletedIPList(ipListId int64) bool {
	deletedIPListLocker.RLock()
	_, ok := deletedIPListIdMap[ipListId]
	deletedIPListLocker.RUnlock()
	return ok
}
