package iplibrary

import (
	"github.com/TeaOSLab/EdgeCommon/pkg/nodeconfigs"
	"github.com/TeaOSLab/EdgeCommon/pkg/rpc/pb"
	teaconst "github.com/TeaOSLab/EdgeNode/internal/const"
	"github.com/TeaOSLab/EdgeNode/internal/events"
	"github.com/TeaOSLab/EdgeNode/internal/goman"
	"github.com/TeaOSLab/EdgeNode/internal/remotelogs"
	"github.com/TeaOSLab/EdgeNode/internal/rpc"
	"github.com/TeaOSLab/EdgeNode/internal/utils"
	"github.com/TeaOSLab/EdgeNode/internal/waf"
	"github.com/TeaOSLab/EdgeNode/internal/zero"
	"github.com/iwind/TeaGo/Tea"
	"sync"
	"time"
)

var SharedIPListManager = NewIPListManager()
var IPListUpdateNotify = make(chan bool, 1)

func init() {
	if !teaconst.IsMain {
		return
	}

	events.On(events.EventLoaded, func() {
		goman.New(func() {
			SharedIPListManager.Start()
		})
	})
	events.OnClose(func() {
		SharedIPListManager.Stop()
	})

	var ticker = time.NewTicker(24 * time.Hour)
	goman.New(func() {
		for range ticker.C {
			SharedIPListManager.DeleteExpiredItems()
		}
	})
}

// IPListManager IP名单管理
type IPListManager struct {
	ticker *time.Ticker

	db *IPListDB

	lastVersion   int64
	fetchPageSize int64

	listMap map[int64]*IPList
	locker  sync.Mutex

	isFirstTime bool
}

func NewIPListManager() *IPListManager {
	return &IPListManager{
		fetchPageSize: 5_000,
		listMap:       map[int64]*IPList{},
		isFirstTime:   true,
	}
}

func (this *IPListManager) Start() {
	this.init()

	// 第一次读取
	err := this.loop()
	if err != nil {
		remotelogs.ErrorObject("IP_LIST_MANAGER", err)
	}

	this.ticker = time.NewTicker(60 * time.Second)
	if Tea.IsTesting() {
		this.ticker = time.NewTicker(10 * time.Second)
	}
	var countErrors = 0
	for {
		select {
		case <-this.ticker.C:
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

func (this *IPListManager) Stop() {
	if this.ticker != nil {
		this.ticker.Stop()
	}
}

func (this *IPListManager) init() {
	// 从数据库中当中读取数据
	db, err := NewIPListDB()
	if err != nil {
		remotelogs.Error("IP_LIST_MANAGER", "create ip list local database failed: "+err.Error())
	} else {
		this.db = db

		// 删除本地数据库中过期的条目
		_ = db.DeleteExpiredItems()

		// 本地数据库中最大版本号
		this.lastVersion = db.ReadMaxVersion()

		// 从本地数据库中加载
		var offset int64 = 0
		var size int64 = 2_000
		for {
			items, err := db.ReadItems(offset, size)
			var l = len(items)
			if err != nil {
				remotelogs.Error("IP_LIST_MANAGER", "read ip list from local database failed: "+err.Error())
			} else {
				if l == 0 {
					break
				}
				this.processItems(items, false)
				if int64(l) < size {
					break
				}
			}
			offset += int64(l)
		}
	}
}

func (this *IPListManager) loop() error {
	// 是否同步IP名单
	nodeConfig, _ := nodeconfigs.SharedNodeConfig()
	if nodeConfig != nil && !nodeConfig.EnableIPLists {
		return nil
	}

	// 第一次同步则打印信息
	if this.isFirstTime {
		remotelogs.Println("IP_LIST_MANAGER", "initializing ip items ...")
	}

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

	// 第一次同步则打印信息
	if this.isFirstTime {
		this.isFirstTime = false
		remotelogs.Println("IP_LIST_MANAGER", "finished initializing ip items")
	}

	return nil
}

func (this *IPListManager) fetch() (hasNext bool, err error) {
	rpcClient, err := rpc.SharedRPC()
	if err != nil {
		return false, err
	}
	itemsResp, err := rpcClient.IPItemRPC.ListIPItemsAfterVersion(rpcClient.Context(), &pb.ListIPItemsAfterVersionRequest{
		Version: this.lastVersion,
		Size:    this.fetchPageSize,
	})
	if err != nil {
		if rpc.IsConnError(err) {
			remotelogs.Debug("IP_LIST_MANAGER", "rpc connection error: "+err.Error())
			return false, nil
		}
		return false, err
	}
	var items = itemsResp.IpItems
	if len(items) == 0 {
		return false, nil
	}

	// 保存到本地数据库
	if this.db != nil {
		for _, item := range items {
			err = this.db.AddItem(item)
			if err != nil {
				remotelogs.Error("IP_LIST_MANAGER", "insert item to local database failed: "+err.Error())
			}
		}
	}

	this.processItems(items, true)

	return true, nil
}

func (this *IPListManager) FindList(listId int64) *IPList {
	this.locker.Lock()
	var list = this.listMap[listId]
	this.locker.Unlock()
	return list
}

func (this *IPListManager) DeleteExpiredItems() {
	if this.db != nil {
		_ = this.db.DeleteExpiredItems()
	}
}

// 处理IP条目
func (this *IPListManager) processItems(items []*pb.IPItem, fromRemote bool) {
	var changedLists = map[*IPList]zero.Zero{}
	for _, item := range items {
		// 调试
		if Tea.IsTesting() {
			this.debugItem(item)
		}

		var list *IPList
		// TODO 实现节点专有List
		if item.ServerId > 0 { // 服务专有List
			switch item.ListType {
			case "black":
				list = SharedServerListManager.FindBlackList(item.ServerId, true)
			case "white":
				list = SharedServerListManager.FindWhiteList(item.ServerId, true)
			}
		} else if item.IsGlobal { // 全局List
			switch item.ListType {
			case "black":
				list = GlobalBlackIPList
			case "white":
				list = GlobalWhiteIPList
			}
		} else { // 其他List
			this.locker.Lock()
			list = this.listMap[item.ListId]
			this.locker.Unlock()
		}
		if list == nil {
			list = NewIPList()
			this.locker.Lock()
			this.listMap[item.ListId] = list
			this.locker.Unlock()
		}

		changedLists[list] = zero.New()

		if item.IsDeleted {
			list.Delete(uint64(item.Id))

			// 从WAF名单中删除
			waf.SharedIPBlackList.RemoveIP(item.IpFrom, item.ServerId, fromRemote)

			// 操作事件
			if fromRemote {
				SharedActionManager.DeleteItem(item.ListType, item)
			}

			continue
		}

		list.AddDelay(&IPItem{
			Id:         uint64(item.Id),
			Type:       item.Type,
			IPFrom:     utils.IP2Long(item.IpFrom),
			IPTo:       utils.IP2Long(item.IpTo),
			ExpiredAt:  item.ExpiredAt,
			EventLevel: item.EventLevel,
		})

		// 事件操作
		if fromRemote {
			SharedActionManager.DeleteItem(item.ListType, item)
			SharedActionManager.AddItem(item.ListType, item)
		}
	}

	for changedList := range changedLists {
		changedList.Sort()
	}

	if fromRemote {
		var latestVersion = items[len(items)-1].Version
		if latestVersion > this.lastVersion {
			this.lastVersion = latestVersion
		}
	}
}

// 调试IP信息
func (this *IPListManager) debugItem(item *pb.IPItem) {
	if item.IsDeleted {
		remotelogs.Debug("IP_ITEM_DEBUG", "delete '"+item.IpFrom+"'")
	} else {
		remotelogs.Debug("IP_ITEM_DEBUG", "add '"+item.IpFrom+"'")
	}
}
