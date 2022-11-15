// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved. Official site: https://goedge.cn .

package nodes

import (
	"sync"
)

var SharedUserManager = NewUserManager()

type User struct {
	ServersEnabled bool
}

type UserManager struct {
	userMap map[int64]*User // id => *User

	locker sync.RWMutex
}

func NewUserManager() *UserManager {
	return &UserManager{
		userMap: map[int64]*User{},
	}
}

func (this *UserManager) UpdateUserServersIsEnabled(userId int64, isEnabled bool) {
	this.locker.Lock()
	u, ok := this.userMap[userId]
	if ok {
		u.ServersEnabled = isEnabled
	} else {
		u = &User{ServersEnabled: isEnabled}
		this.userMap[userId] = u
	}
	this.locker.Unlock()
}

func (this *UserManager) CheckUserServersIsEnabled(userId int64) (isEnabled bool) {
	if userId <= 0 {
		return true
	}
	
	this.locker.RLock()
	u, ok := this.userMap[userId]
	if ok {
		isEnabled = u.ServersEnabled
	} else {
		isEnabled = true
	}
	this.locker.RUnlock()
	return
}
