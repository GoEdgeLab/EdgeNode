package caches

import (
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs"
	"github.com/iwind/TeaGo/logs"
	"strconv"
	"sync"
)

var SharedManager = NewManager()

type Manager struct {
	policyMap  map[int64]*serverconfigs.HTTPCachePolicy // policyId => []*Policy
	storageMap map[int64]StorageInterface               // policyId => *Storage
	locker     sync.RWMutex
}

func NewManager() *Manager {
	return &Manager{
		policyMap:  map[int64]*serverconfigs.HTTPCachePolicy{},
		storageMap: map[int64]StorageInterface{},
	}
}

// 重新设置策略
func (this *Manager) UpdatePolicies(newPolicies []*serverconfigs.HTTPCachePolicy) {
	this.locker.Lock()
	defer this.locker.Unlock()

	newPolicyIds := []int64{}
	for _, policy := range newPolicies {
		newPolicyIds = append(newPolicyIds, policy.Id)
	}

	// 停止旧有的
	for _, oldPolicy := range this.policyMap {
		if !this.containsInt64(newPolicyIds, oldPolicy.Id) {
			logs.Println("[CACHE]remove policy", strconv.FormatInt(oldPolicy.Id, 10))
			delete(this.policyMap, oldPolicy.Id)
			storage, ok := this.storageMap[oldPolicy.Id]
			if ok {
				storage.Stop()
				delete(this.storageMap, oldPolicy.Id)
			}
		}
	}

	// 启动新的
	for _, newPolicy := range newPolicies {
		_, ok := this.policyMap[newPolicy.Id]
		if !ok {
			logs.Println("[CACHE]add policy", strconv.FormatInt(newPolicy.Id, 10))
		}

		// 初始化
		err := newPolicy.Init()
		if err != nil {
			logs.Println("[CACHE]UpdatePolicies: init policy error: " + err.Error())
			continue
		}
		this.policyMap[newPolicy.Id] = newPolicy
	}

	// 启动存储管理
	for _, policy := range this.policyMap {
		storage, ok := this.storageMap[policy.Id]
		if !ok {
			storage := this.NewStorageWithPolicy(policy)
			if storage == nil {
				logs.Println("[CACHE]can not find storage type '" + policy.Type + "'")
				continue
			}
			err := storage.Init()
			if err != nil {
				logs.Println("[CACHE]UpdatePolicies: init storage failed: " + err.Error())
				continue
			}
			this.storageMap[policy.Id] = storage
		} else {
			// 检查policy是否有变化
			if !storage.Policy().IsSame(policy) {
				logs.Println("[CACHE]policy " + strconv.FormatInt(policy.Id, 10) + " changed")

				// 停止老的
				storage.Stop()
				delete(this.storageMap, policy.Id)

				// 启动新的
				storage := this.NewStorageWithPolicy(policy)
				if storage == nil {
					logs.Println("[CACHE]can not find storage type '" + policy.Type + "'")
					continue
				}
				err := storage.Init()
				if err != nil {
					logs.Println("[CACHE]UpdatePolicies: init storage failed: " + err.Error())
					continue
				}
				this.storageMap[policy.Id] = storage
			}
		}
	}
}

// 获取Policy信息
func (this *Manager) FindPolicy(policyId int64) *serverconfigs.HTTPCachePolicy {
	this.locker.RLock()
	defer this.locker.RUnlock()

	p, _ := this.policyMap[policyId]
	return p
}

// 根据策略ID查找存储
func (this *Manager) FindStorageWithPolicy(policyId int64) StorageInterface {
	this.locker.RLock()
	defer this.locker.RUnlock()

	storage, _ := this.storageMap[policyId]
	return storage
}

// 根据策略获取存储对象
func (this *Manager) NewStorageWithPolicy(policy *serverconfigs.HTTPCachePolicy) StorageInterface {
	switch policy.Type {
	case serverconfigs.CachePolicyStorageFile:
		return NewFileStorage(policy)
	case serverconfigs.CachePolicyStorageMemory:
		return nil // TODO 暂时返回nil
	}
	return nil
}

// 可判断一组数字中是否包含某数
func (this *Manager) containsInt64(values []int64, value int64) bool {
	for _, v := range values {
		if v == value {
			return true
		}
	}
	return false
}
