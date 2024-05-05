package iplibrary

import (
	"github.com/TeaOSLab/EdgeCommon/pkg/iputils"
	"github.com/TeaOSLab/EdgeCommon/pkg/nodeconfigs"
	"github.com/TeaOSLab/EdgeCommon/pkg/rpc/pb"
	teaconst "github.com/TeaOSLab/EdgeNode/internal/const"
	"github.com/TeaOSLab/EdgeNode/internal/events"
	"github.com/TeaOSLab/EdgeNode/internal/goman"
	"github.com/TeaOSLab/EdgeNode/internal/remotelogs"
	"github.com/TeaOSLab/EdgeNode/internal/rpc"
	"github.com/TeaOSLab/EdgeNode/internal/trackers"
	"github.com/TeaOSLab/EdgeNode/internal/utils/idles"
	"github.com/TeaOSLab/EdgeNode/internal/waf"
	"github.com/TeaOSLab/EdgeNode/internal/zero"
	"github.com/iwind/TeaGo/Tea"
	"github.com/iwind/TeaGo/types"
	"os"
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
		idles.RunTicker(ticker, func() {
			SharedIPListManager.DeleteExpiredItems()
		})
	})
}

// IPListManager IP名单管理
type IPListManager struct {
	ticker *time.Ticker

	db IPListDB

	lastVersion   int64
	fetchPageSize int64

	listMap map[int64]*IPList
	mu      sync.RWMutex

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
	this.Init()

	// 第一次读取
	err := this.Loop()
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
		err = this.Loop()
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

func (this *IPListManager) Init() {
	// 从数据库中当中读取数据
	// 检查sqlite文件是否存在，以便决定使用sqlite还是kv
	var sqlitePath = Tea.Root + "/data/ip_list.db"
	_, sqliteErr := os.Stat(sqlitePath)

	var db IPListDB
	var err error
	if sqliteErr == nil || !teaconst.EnableKVCacheStore {
		db, err = NewSQLiteIPList()
	} else {
		db, err = NewKVIPList()
	}

	if err != nil {
		remotelogs.Error("IP_LIST_MANAGER", "create ip list local database failed: "+err.Error())
	} else {
		this.db = db

		// 删除本地数据库中过期的条目
		_ = db.DeleteExpiredItems()

		// 本地数据库中最大版本号
		this.lastVersion, err = db.ReadMaxVersion()
		if err != nil {
			remotelogs.Error("IP_LIST_MANAGER", "find max version failed: "+err.Error())
			this.lastVersion = 0
		}
		remotelogs.Println("IP_LIST_MANAGER", "starting from '"+db.Name()+"' version '"+types.String(this.lastVersion)+"' ...")

		// 从本地数据库中加载
		var offset int64 = 0
		var size int64 = 2_000

		var tr = trackers.Begin("IP_LIST_MANAGER:load")
		defer tr.End()

		for {
			items, goNext, readErr := db.ReadItems(offset, size)
			var l = len(items)
			if readErr != nil {
				remotelogs.Error("IP_LIST_MANAGER", "read ip list from local database failed: "+readErr.Error())
			} else {
				this.processItems(items, false)
				if !goNext {
					break
				}
			}
			offset += int64(l)
		}
	}
}

func (this *IPListManager) Loop() error {
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

	// 更新版本号
	defer func() {
		if itemsResp.Version > this.lastVersion {
			this.lastVersion = itemsResp.Version
			err = this.db.UpdateMaxVersion(itemsResp.Version)
			if err != nil {
				remotelogs.Error("IP_LIST_MANAGER", "update max version to database: "+err.Error())
			}
		}
	}()

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
	this.mu.RLock()
	var list = this.listMap[listId]
	this.mu.RUnlock()

	return list
}

func (this *IPListManager) DeleteExpiredItems() {
	if this.db != nil {
		_ = this.db.DeleteExpiredItems()
	}
}

func (this *IPListManager) ListMap() map[int64]*IPList {
	return this.listMap
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
			this.mu.Lock()
			list = this.listMap[item.ListId]
			this.mu.Unlock()
		}
		if list == nil {
			list = NewIPList()
			this.mu.Lock()
			this.listMap[item.ListId] = list
			this.mu.Unlock()
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
			IPFrom:     iputils.ToBytes(item.IpFrom),
			IPTo:       iputils.ToBytes(item.IpTo),
			ExpiredAt:  item.ExpiredAt,
			EventLevel: item.EventLevel,
		})

		// 事件操作
		if fromRemote {
			SharedActionManager.DeleteItem(item.ListType, item)
			SharedActionManager.AddItem(item.ListType, item)
		}
	}

	if len(changedLists) > 0 {
		for changedList := range changedLists {
			changedList.Sort()
		}
	}
}

// 调试IP信息
func (this *IPListManager) debugItem(item *pb.IPItem) {
	var ipRange = item.IpFrom
	if len(item.IpTo) > 0 {
		ipRange += " - " + item.IpTo
	}

	if item.IsDeleted {
		remotelogs.Debug("IP_ITEM_DEBUG", "delete '"+ipRange+"'")
	} else {
		remotelogs.Debug("IP_ITEM_DEBUG", "add '"+ipRange+"'")
	}
}
