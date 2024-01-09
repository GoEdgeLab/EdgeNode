// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package waf

import (
	"encoding/json"
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs/firewallconfigs"
	"github.com/TeaOSLab/EdgeNode/internal/conns"
	teaconst "github.com/TeaOSLab/EdgeNode/internal/const"
	"github.com/TeaOSLab/EdgeNode/internal/events"
	"github.com/TeaOSLab/EdgeNode/internal/firewalls"
	"github.com/TeaOSLab/EdgeNode/internal/utils/expires"
	"github.com/TeaOSLab/EdgeNode/internal/utils/fasttime"
	"github.com/iwind/TeaGo/Tea"
	"github.com/iwind/TeaGo/maps"
	"github.com/iwind/TeaGo/types"
	"os"
	"sync"
	"sync/atomic"
)

var SharedIPWhiteList = NewIPList(IPListTypeAllow)
var SharedIPBlackList = NewIPList(IPListTypeDeny)

type IPListType = string

const (
	IPListTypeAllow IPListType = "allow"
	IPListTypeDeny  IPListType = "deny"
)

const IPTypeAll = "*"

func init() {
	if !teaconst.IsMain {
		return
	}

	var cacheFile = Tea.Root + "/data/waf_white_list.cache"

	// save
	events.On(events.EventTerminated, func() {
		_ = SharedIPWhiteList.Save(cacheFile)
	})

	// load
	go func() {
		if !Tea.IsTesting() {
			_ = SharedIPWhiteList.Load(cacheFile)
			_ = os.Remove(cacheFile)
		}
	}()
}

// IPList IP列表管理
type IPList struct {
	expireList *expires.List
	ipMap      map[string]uint64 // ip info => id
	idMap      map[uint64]string // id => ip info
	listType   IPListType

	id     uint64
	locker sync.RWMutex

	lastIP   string // 加入到 recordIPTaskChan 之前尽可能去重
	lastTime int64
}

// NewIPList 获取新对象
func NewIPList(listType IPListType) *IPList {
	var list = &IPList{
		ipMap:    map[string]uint64{},
		idMap:    map[uint64]string{},
		listType: listType,
	}

	var e = expires.NewList()
	list.expireList = e

	e.OnGC(func(itemId uint64) {
		list.remove(itemId) // TODO 使用异步，防止阻塞GC
	})

	return list
}

// Add 添加IP
func (this *IPList) Add(ipType string, scope firewallconfigs.FirewallScope, serverId int64, ip string, expiresAt int64) {
	switch scope {
	case firewallconfigs.FirewallScopeGlobal:
		ip = "*@" + ip + "@" + ipType
	case firewallconfigs.FirewallScopeService:
		ip = types.String(serverId) + "@" + ip + "@" + ipType
	default:
		ip = "*@" + ip + "@" + ipType
	}

	var id = this.nextId()
	this.expireList.Add(id, expiresAt)
	this.locker.Lock()

	// 删除以前
	oldId, ok := this.ipMap[ip]
	if ok {
		delete(this.idMap, oldId)
		this.expireList.Remove(oldId)
	}

	this.ipMap[ip] = id
	this.idMap[id] = ip
	this.locker.Unlock()
}

// RecordIP 记录IP
func (this *IPList) RecordIP(ipType string,
	scope firewallconfigs.FirewallScope,
	serverId int64,
	ip string,
	expiresAt int64,
	policyId int64,
	useLocalFirewall bool,
	groupId int64,
	setId int64,
	reason string) {
	this.Add(ipType, scope, serverId, ip, expiresAt)

	if this.listType == IPListTypeDeny {
		// 作用域
		var scopeServerId int64
		if scope == firewallconfigs.FirewallScopeService {
			scopeServerId = serverId
		}

		// 加入队列等待上传
		if this.lastIP != ip || fasttime.Now().Unix()-this.lastTime > 3 /** 3秒外才允许重复添加 **/ {
			select {
			case recordIPTaskChan <- &recordIPTask{
				ip:                            ip,
				listId:                        firewallconfigs.GlobalListId,
				expiresAt:                     expiresAt,
				level:                         firewallconfigs.DefaultEventLevel,
				serverId:                      scopeServerId,
				sourceServerId:                serverId,
				sourceHTTPFirewallPolicyId:    policyId,
				sourceHTTPFirewallRuleGroupId: groupId,
				sourceHTTPFirewallRuleSetId:   setId,
				reason:                        reason,
			}:
				this.lastIP = ip
				this.lastTime = fasttime.Now().Unix()
			default:
			}

			// 使用本地防火墙
			if useLocalFirewall {
				firewalls.DropTemporaryTo(ip, expiresAt)
			}
		}

		// 关闭此IP相关连接
		conns.SharedMap.CloseIPConns(ip)
	}
}

// Contains 判断是否有某个IP
func (this *IPList) Contains(ipType string, scope firewallconfigs.FirewallScope, serverId int64, ip string) bool {
	switch scope {
	case firewallconfigs.FirewallScopeGlobal:
		ip = "*@" + ip + "@" + ipType
	case firewallconfigs.FirewallScopeService:
		ip = types.String(serverId) + "@" + ip + "@" + ipType
	default:
		ip = "*@" + ip + "@" + ipType
	}

	this.locker.RLock()
	_, ok := this.ipMap[ip]
	this.locker.RUnlock()
	return ok
}

// ContainsExpires 判断是否有某个IP，并返回过期时间
func (this *IPList) ContainsExpires(ipType string, scope firewallconfigs.FirewallScope, serverId int64, ip string) (expiresAt int64, ok bool) {
	switch scope {
	case firewallconfigs.FirewallScopeGlobal:
		ip = "*@" + ip + "@" + ipType
	case firewallconfigs.FirewallScopeService:
		ip = types.String(serverId) + "@" + ip + "@" + ipType
	default:
		ip = "*@" + ip + "@" + ipType
	}

	this.locker.RLock()
	id, ok := this.ipMap[ip]
	if ok {
		expiresAt = this.expireList.ExpiresAt(id)
	}
	this.locker.RUnlock()
	return expiresAt, ok
}

// RemoveIP 删除IP
func (this *IPList) RemoveIP(ip string, serverId int64, shouldExecute bool) {
	this.locker.Lock()

	{
		var key = "*@" + ip + "@" + IPTypeAll
		id, ok := this.ipMap[key]
		if ok {
			delete(this.ipMap, key)
			delete(this.idMap, id)

			this.expireList.Remove(id)
		}
	}

	if serverId > 0 {
		var key = types.String(serverId) + "@" + ip + "@" + IPTypeAll
		id, ok := this.ipMap[key]
		if ok {
			delete(this.ipMap, key)
			delete(this.idMap, id)

			this.expireList.Remove(id)
		}
	}

	this.locker.Unlock()

	// 从本地防火墙中删除
	if shouldExecute {
		_ = firewalls.Firewall().RemoveSourceIP(ip)
	}
}

// Save to local file
func (this *IPList) Save(path string) error {
	var itemMaps = []maps.Map{} // [ {ip info, expiresAt }, ... ]
	this.locker.Lock()
	defer this.locker.Unlock()

	// prevent too many items
	if len(this.ipMap) > 100_000 {
		return nil
	}

	for ipInfo, id := range this.ipMap {
		var expiresAt = this.expireList.ExpiresAt(id)
		if expiresAt <= 0 {
			continue
		}
		itemMaps = append(itemMaps, maps.Map{
			"ip":        ipInfo,
			"expiresAt": expiresAt,
		})
	}

	itemMapsJSON, err := json.Marshal(itemMaps)
	if err != nil {
		return err
	}
	return os.WriteFile(path, itemMapsJSON, 0666)
}

// Load from local file
func (this *IPList) Load(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if len(data) == 0 {
		return nil
	}

	var itemMaps = []maps.Map{}
	err = json.Unmarshal(data, &itemMaps)
	if err != nil {
		return err
	}

	this.locker.Lock()
	defer this.locker.Unlock()

	for _, itemMap := range itemMaps {
		var ip = itemMap.GetString("ip")
		var expiresAt = itemMap.GetInt64("expiresAt")
		if len(ip) == 0 || expiresAt < fasttime.Now().Unix()+10 /** seconds **/ {
			continue
		}

		var id = this.nextId()
		this.expireList.Add(id, expiresAt)

		this.ipMap[ip] = id
		this.idMap[id] = ip
	}

	return nil
}

// IPMap get ipMap
func (this *IPList) IPMap() map[string]uint64 {
	this.locker.RLock()
	defer this.locker.RUnlock()
	return this.ipMap
}

// IdMap get idMap
func (this *IPList) IdMap() map[uint64]string {
	this.locker.RLock()
	defer this.locker.RUnlock()
	return this.idMap
}

func (this *IPList) remove(id uint64) {
	this.locker.Lock()
	ip, ok := this.idMap[id]
	if ok {
		ipId, ok := this.ipMap[ip]
		if ok && ipId == id {
			delete(this.ipMap, ip)
		}
		delete(this.idMap, id)
	}
	this.locker.Unlock()
}

func (this *IPList) nextId() uint64 {
	return atomic.AddUint64(&this.id, 1)
}
