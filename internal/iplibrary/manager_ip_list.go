package iplibrary

import (
	"github.com/TeaOSLab/EdgeCommon/pkg/rpc/pb"
	"github.com/TeaOSLab/EdgeNode/internal/events"
	"github.com/TeaOSLab/EdgeNode/internal/logs"
	"github.com/TeaOSLab/EdgeNode/internal/rpc"
	"github.com/iwind/TeaGo/Tea"
	"sync"
	"time"
)

var SharedIPListManager = NewIPListManager()
var IPListUpdateNotify = make(chan bool, 1)

func init() {
	events.On(events.EventStart, func() {
		go SharedIPListManager.Start()
	})
}

// IP名单管理
type IPListManager struct {
	// 缓存文件
	// 每行一个数据：id|from|to|expiredAt
	cacheFile string

	version  int64
	pageSize int64

	listMap map[int64]*IPList
	locker  sync.Mutex
}

func NewIPListManager() *IPListManager {
	return &IPListManager{
		cacheFile: Tea.Root + "/configs/ip_list.cache",
		pageSize:  1000,
		listMap:   map[int64]*IPList{},
	}
}

func (this *IPListManager) Start() {
	// TODO 从缓存当中读取数据

	// 第一次读取
	err := this.loop()
	if err != nil {
		logs.Println("IP_LIST_MANAGER", err.Error())
	}

	ticker := time.NewTicker(60 * time.Second) // TODO 未来改成可以手动触发IP变更事件
	events.On(events.EventQuit, func() {
		ticker.Stop()
	})
	countErrors := 0
	for {
		select {
		case <-ticker.C:
		case <-IPListUpdateNotify:
		}
		err := this.loop()
		if err != nil {
			countErrors++

			logs.Println("IP_LIST_MANAGER", err.Error())

			// 连续错误小于3次的我们立即重试
			if countErrors <= 3 {
				select {
				case IPListUpdateNotify <- true:
				default:
				}
			}
		} else {
			countErrors = 0
		}
	}
}

func (this *IPListManager) loop() error {
	for {
		hasNext, err := this.fetch()
		if err != nil {
			return err
		}
		if !hasNext {
			break
		}
	}

	// TODO 写入到缓存当中

	return nil
}

func (this *IPListManager) fetch() (hasNext bool, err error) {
	rpcClient, err := rpc.SharedRPC()
	if err != nil {
		return false, err
	}
	itemsResp, err := rpcClient.IPItemRPC().ListIPItemsAfterVersion(rpcClient.Context(), &pb.ListIPItemsAfterVersionRequest{
		Version: this.version,
		Size:    this.pageSize,
	})
	if err != nil {
		return false, err
	}
	items := itemsResp.IpItems
	if len(items) == 0 {
		return false, nil
	}
	this.locker.Lock()
	for _, item := range items {
		list, ok := this.listMap[item.ListId]
		if !ok {
			list = NewIPList()
			this.listMap[item.ListId] = list
		}
		if item.IsDeleted {
			list.Delete(item.Id)
			continue
		}
		list.Add(&IPItem{
			Id:        item.Id,
			IPFrom:    IP2Long(item.IpFrom),
			IPTo:      IP2Long(item.IpTo),
			ExpiredAt: item.ExpiredAt,
		})
	}
	this.locker.Unlock()
	this.version = items[len(items)-1].Version

	return true, nil
}

func (this *IPListManager) FindList(listId int64) *IPList {
	this.locker.Lock()
	list, _ := this.listMap[listId]
	this.locker.Unlock()
	return list
}
