// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package nodes

import (
	"github.com/TeaOSLab/EdgeNode/internal/ttlcache"
	"github.com/TeaOSLab/EdgeNode/internal/utils"
	"github.com/iwind/TeaGo/types"
	"strings"
	"sync"
	"time"
)

var sharedCNAMEManager = NewServerCNAMEManager()

// ServerCNAMEManager 服务CNAME管理
// TODO 需要自动更新缓存里的记录
type ServerCNAMEManager struct {
	ttlCache *ttlcache.Cache

	locker sync.Mutex
}

func NewServerCNAMEManager() *ServerCNAMEManager {
	return &ServerCNAMEManager{
		ttlCache: ttlcache.NewCache(),
	}
}

func (this *ServerCNAMEManager) Lookup(domain string) string {
	if len(domain) == 0 {
		return ""
	}

	var item = this.ttlCache.Read(domain)
	if item != nil {
		return types.String(item.Value)
	}

	cname, _ := utils.LookupCNAME(domain)
	if len(cname) > 0 {
		cname = strings.TrimSuffix(cname, ".")
	}

	this.ttlCache.Write(domain, cname, time.Now().Unix()+600)

	return cname
}
