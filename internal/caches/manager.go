package caches

import (
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs"
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs/shared"
	"github.com/TeaOSLab/EdgeNode/internal/remotelogs"
	"github.com/iwind/TeaGo/lists"
	"strconv"
	"sync"
)

var SharedManager = NewManager()

// Manager 缓存策略管理器
type Manager struct {
	// 全局配置
	MaxDiskCapacity   *shared.SizeCapacity
	MaxMemoryCapacity *shared.SizeCapacity

	policyMap  map[int64]*serverconfigs.HTTPCachePolicy // policyId => []*Policy
	storageMap map[int64]StorageInterface               // policyId => *Storage
	locker     sync.RWMutex
}

// NewManager 获取管理器对象
func NewManager() *Manager {
	return &Manager{
		policyMap:  map[int64]*serverconfigs.HTTPCachePolicy{},
		storageMap: map[int64]StorageInterface{},
	}
}

// UpdatePolicies 重新设置策略
func (this *Manager) UpdatePolicies(newPolicies []*serverconfigs.HTTPCachePolicy) {
	this.locker.Lock()
	defer this.locker.Unlock()

	newPolicyIds := []int64{}
	for _, policy := range newPolicies {
		newPolicyIds = append(newPolicyIds, policy.Id)
	}

	// 停止旧有的
	for _, oldPolicy := range this.policyMap {
		if !lists.ContainsInt64(newPolicyIds, oldPolicy.Id) {
			remotelogs.Error("CACHE", "remove policy "+strconv.FormatInt(oldPolicy.Id, 10))
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
			remotelogs.Println("CACHE", "add policy "+strconv.FormatInt(newPolicy.Id, 10))
		}

		// 初始化
		err := newPolicy.Init()
		if err != nil {
			remotelogs.Error("CACHE", "UpdatePolicies: init policy error: "+err.Error())
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
				remotelogs.Error("CACHE", "can not find storage type '"+policy.Type+"'")
				continue
			}
			err := storage.Init()
			if err != nil {
				remotelogs.Error("CACHE", "UpdatePolicies: init storage failed: "+err.Error())
				continue
			}
			this.storageMap[policy.Id] = storage
		} else {
			// 检查policy是否有变化
			if !storage.Policy().IsSame(policy) {
				remotelogs.Println("CACHE", "policy "+strconv.FormatInt(policy.Id, 10)+" changed")

				// 停止老的
				storage.Stop()
				delete(this.storageMap, policy.Id)

				// 启动新的
				storage := this.NewStorageWithPolicy(policy)
				if storage == nil {
					remotelogs.Error("CACHE", "can not find storage type '"+policy.Type+"'")
					continue
				}
				err := storage.Init()
				if err != nil {
					remotelogs.Error("CACHE", "UpdatePolicies: init storage failed: "+err.Error())
					continue
				}
				this.storageMap[policy.Id] = storage
			}
		}
	}
}

// FindPolicy 获取Policy信息
func (this *Manager) FindPolicy(policyId int64) *serverconfigs.HTTPCachePolicy {
	this.locker.RLock()
	defer this.locker.RUnlock()

	p, _ := this.policyMap[policyId]
	return p
}

// FindStorageWithPolicy 根据策略ID查找存储
func (this *Manager) FindStorageWithPolicy(policyId int64) StorageInterface {
	this.locker.RLock()
	defer this.locker.RUnlock()

	storage, _ := this.storageMap[policyId]
	return storage
}

// NewStorageWithPolicy 根据策略获取存储对象
func (this *Manager) NewStorageWithPolicy(policy *serverconfigs.HTTPCachePolicy) StorageInterface {
	switch policy.Type {
	case serverconfigs.CachePolicyStorageFile:
		return NewFileStorage(policy)
	case serverconfigs.CachePolicyStorageMemory:
		return NewMemoryStorage(policy)
	}
	return nil
}

// TotalDiskSize 消耗的磁盘尺寸
func (this *Manager) TotalDiskSize() int64 {
	this.locker.RLock()
	defer this.locker.RUnlock()

	total := int64(0)
	for _, storage := range this.storageMap {
		total += storage.TotalDiskSize()
	}
	return total
}

// TotalMemorySize 消耗的内存尺寸
func (this *Manager) TotalMemorySize() int64 {
	this.locker.RLock()
	defer this.locker.RUnlock()

	total := int64(0)
	for _, storage := range this.storageMap {
		total += storage.TotalMemorySize()
	}
	return total
}
