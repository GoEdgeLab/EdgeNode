// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package waf

import (
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs/firewallconfigs"
	"github.com/TeaOSLab/EdgeNode/internal/conns"
	"github.com/TeaOSLab/EdgeNode/internal/firewalls"
	"github.com/TeaOSLab/EdgeNode/internal/utils/expires"
	"github.com/TeaOSLab/EdgeNode/internal/utils/fasttime"
	"github.com/iwind/TeaGo/types"
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

// IPList IP列表管理
type IPList struct {
	expireList *expires.List
	ipMap      map[string]uint64 // ip => id
	idMap      map[uint64]string // id => ip
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

	e := expires.NewList()
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
