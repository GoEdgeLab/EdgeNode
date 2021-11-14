package iplibrary

import (
	"github.com/TeaOSLab/EdgeCommon/pkg/rpc/pb"
	"github.com/TeaOSLab/EdgeNode/internal/events"
	"github.com/TeaOSLab/EdgeNode/internal/remotelogs"
	"github.com/TeaOSLab/EdgeNode/internal/rpc"
	"github.com/TeaOSLab/EdgeNode/internal/utils"
	"github.com/TeaOSLab/EdgeNode/internal/waf"
	"github.com/iwind/TeaGo/Tea"
	"github.com/iwind/TeaGo/types"
	"io/ioutil"
	"os"
	"sync"
	"time"
)

var SharedIPListManager = NewIPListManager()
var IPListUpdateNotify = make(chan bool, 1)

func init() {
	events.On(events.EventLoaded, func() {
		go SharedIPListManager.Start()
	})
}

var versionCacheFile = "ip_list_version.cache"

// IPListManager IP名单管理
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
		pageSize:  500,
		listMap:   map[int64]*IPList{},
	}
}

func (this *IPListManager) Start() {
	// TODO 从缓存当中读取数据

	// 从缓存中读取位置
	this.version = this.readLocalVersion()

	// 第一次读取
	err := this.loop()
	if err != nil {
		remotelogs.ErrorObject("IP_LIST_MANAGER", err)
	}

	ticker := time.NewTicker(60 * time.Second)
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

			remotelogs.ErrorObject("IP_LIST_MANAGER", err)

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
		time.Sleep(1 * time.Second)
	}

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
	var changedLists = map[*IPList]bool{}
	for _, item := range items {
		list, ok := this.listMap[item.ListId]
		if !ok {
			list = NewIPList()
			this.listMap[item.ListId] = list
		}

		changedLists[list] = true

		if item.IsDeleted {
			list.Delete(item.Id)

			// 从临时名单中删除
			if len(item.IpFrom) > 0 && len(item.IpTo) == 0 {
				switch item.ListType {
				case "black":
					waf.SharedIPBlackList.RemoveIP(item.IpFrom)
				case "white":
					waf.SharedIPWhiteList.RemoveIP(item.IpFrom)
				}
			}

			// 操作事件
			SharedActionManager.DeleteItem(item.ListType, item)

			continue
		}

		list.AddDelay(&IPItem{
			Id:         item.Id,
			Type:       item.Type,
			IPFrom:     utils.IP2Long(item.IpFrom),
			IPTo:       utils.IP2Long(item.IpTo),
			ExpiredAt:  item.ExpiredAt,
			EventLevel: item.EventLevel,
		})

		// 事件操作
		SharedActionManager.DeleteItem(item.ListType, item)
		SharedActionManager.AddItem(item.ListType, item)
	}

	for changedList := range changedLists {
		changedList.Sort()
	}

	this.locker.Unlock()
	this.version = items[len(items)-1].Version

	// 写入版本号到缓存当中
	this.updateLocalVersion(this.version)

	return true, nil
}

func (this *IPListManager) FindList(listId int64) *IPList {
	this.locker.Lock()
	list, _ := this.listMap[listId]
	this.locker.Unlock()
	return list
}

func (this *IPListManager) readLocalVersion() int64 {
	data, err := ioutil.ReadFile(Tea.ConfigFile(versionCacheFile))
	if err != nil || len(data) == 0 {
		return 0
	}
	return types.Int64(string(data))
}

func (this *IPListManager) updateLocalVersion(version int64) {
	fp, err := os.OpenFile(Tea.ConfigFile(versionCacheFile), os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0666)
	if err != nil {
		remotelogs.Warn("IP_LIST", "write local version cache failed: "+err.Error())
		return
	}
	_, err = fp.WriteString(types.String(version))
	if err != nil {
		remotelogs.Warn("IP_LIST", "write local version cache failed: "+err.Error())
		return
	}
	_ = fp.Close()
}
