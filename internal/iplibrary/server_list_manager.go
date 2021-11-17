// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package iplibrary

import "sync"

var SharedServerListManager = NewServerListManager()

// ServerListManager 服务相关名单
type ServerListManager struct {
	whiteMap map[int64]*IPList // serverId => *List
	blackMap map[int64]*IPList // serverId => *List

	locker sync.RWMutex
}

func NewServerListManager() *ServerListManager {
	return &ServerListManager{
		whiteMap: map[int64]*IPList{},
		blackMap: map[int64]*IPList{},
	}
}

func (this *ServerListManager) FindWhiteList(serverId int64, autoCreate bool) *IPList {
	this.locker.RLock()
	list, ok := this.whiteMap[serverId]
	this.locker.RUnlock()
	if ok {
		return list
	}

	if autoCreate {
		list = NewIPList()
		this.locker.Lock()
		this.whiteMap[serverId] = list
		this.locker.Unlock()

		return list
	}
	return nil
}

func (this *ServerListManager) FindBlackList(serverId int64, autoCreate bool) *IPList {
	this.locker.RLock()
	list, ok := this.blackMap[serverId]
	this.locker.RUnlock()
	if ok {
		return list
	}

	if autoCreate {
		list = NewIPList()
		this.locker.Lock()
		this.blackMap[serverId] = list
		this.locker.Unlock()

		return list
	}

	return nil
}
