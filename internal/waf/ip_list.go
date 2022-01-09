// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package waf

import (
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs/firewallconfigs"
	"github.com/TeaOSLab/EdgeNode/internal/firewalls"
	"github.com/TeaOSLab/EdgeNode/internal/utils/expires"
	"github.com/iwind/TeaGo/types"
	"sync"
	"sync/atomic"
	"time"
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
	ipMap      map[string]int64 // ip => id
	idMap      map[int64]string // id => ip
	listType   IPListType

	id     int64
	locker sync.RWMutex
}

// NewIPList 获取新对象
func NewIPList(listType IPListType) *IPList {
	var list = &IPList{
		ipMap:    map[string]int64{},
		idMap:    map[int64]string{},
		listType: listType,
	}

	e := expires.NewList()
	list.expireList = e

	e.OnGC(func(itemId int64) {
		list.remove(itemId)
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
	setId int64) {
	this.Add(ipType, scope, serverId, ip, expiresAt)

	if this.listType == IPListTypeDeny {
		// 加入队列等待上传
		select {
		case recordIPTaskChan <- &recordIPTask{
			ip:                            ip,
			listId:                        firewallconfigs.GlobalListId,
			expiredAt:                     expiresAt,
			level:                         firewallconfigs.DefaultEventLevel,
			serverId:                      serverId,
			sourceServerId:                serverId,
			sourceHTTPFirewallPolicyId:    policyId,
			sourceHTTPFirewallRuleGroupId: groupId,
			sourceHTTPFirewallRuleSetId:   setId,
		}:
		default:

		}

		// 使用本地防火墙
		if useLocalFirewall {
			var seconds = expiresAt - time.Now().Unix()
			if seconds > 0 {
				// 最大3600，防止误封时间过长
				if seconds > 3600 {
					seconds = 3600
				}
				_ = firewalls.Firewall().DropSourceIP(ip, int(seconds))
			}
		}
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
	defer this.locker.RUnlock()
	_, ok := this.ipMap[ip]
	return ok
}

// RemoveIP 删除IP
func (this *IPList) RemoveIP(ip string, serverId int64, shouldExecute bool) {
	this.locker.Lock()
	delete(this.ipMap, "*@"+ip+"@"+IPTypeAll)
	if serverId > 0 {
		delete(this.ipMap, types.String(serverId)+"@"+ip+"@"+IPTypeAll)
	}
	this.locker.Unlock()

	// 从本地防火墙中删除
	if shouldExecute {
		_ = firewalls.Firewall().RemoveSourceIP(ip)
	}
}

func (this *IPList) remove(id int64) {
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

func (this *IPList) nextId() int64 {
	return atomic.AddInt64(&this.id, 1)
}
