package caches

import (
	"fmt"
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs"
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs/shared"
	teaconst "github.com/TeaOSLab/EdgeNode/internal/const"
	"github.com/TeaOSLab/EdgeNode/internal/events"
	"github.com/TeaOSLab/EdgeNode/internal/remotelogs"
	"github.com/iwind/TeaGo/lists"
	"github.com/iwind/TeaGo/types"
	"golang.org/x/sys/unix"
	"strconv"
	"sync"
)

var SharedManager = NewManager()

func init() {
	if !teaconst.IsMain {
		return
	}

	events.OnClose(func() {
		remotelogs.Println("CACHE", "quiting cache manager")
		SharedManager.UpdatePolicies([]*serverconfigs.HTTPCachePolicy{})
	})
}

// Manager 缓存策略管理器
type Manager struct {
	// 全局配置
	MaxDiskCapacity   *shared.SizeCapacity
	MainDiskDir       string
	SubDiskDirs       []*serverconfigs.CacheDir
	MaxMemoryCapacity *shared.SizeCapacity

	policyMap  map[int64]*serverconfigs.HTTPCachePolicy // policyId => []*Policy
	storageMap map[int64]StorageInterface               // policyId => *Storage
	locker     sync.RWMutex
}

// NewManager 获取管理器对象
func NewManager() *Manager {
	var m = &Manager{
		policyMap:  map[int64]*serverconfigs.HTTPCachePolicy{},
		storageMap: map[int64]StorageInterface{},
	}

	return m
}

// UpdatePolicies 重新设置策略
func (this *Manager) UpdatePolicies(newPolicies []*serverconfigs.HTTPCachePolicy) {
	this.locker.Lock()
	defer this.locker.Unlock()

	var newPolicyIds = []int64{}
	for _, policy := range newPolicies {
		// 使用节点单独的缓存目录
		policy.UpdateDiskDir(this.MainDiskDir, this.SubDiskDirs)

		newPolicyIds = append(newPolicyIds, policy.Id)
	}

	// 停止旧有的
	for _, oldPolicy := range this.policyMap {
		if !lists.ContainsInt64(newPolicyIds, oldPolicy.Id) {
			remotelogs.Println("CACHE", "remove policy "+strconv.FormatInt(oldPolicy.Id, 10))
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
			storage = this.NewStorageWithPolicy(policy)
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
				// 检查是否可以直接修改
				if storage.CanUpdatePolicy(policy) {
					err := policy.Init()
					if err != nil {
						remotelogs.Error("CACHE", "reload policy '"+types.String(policy.Id)+"' failed: init policy failed: "+err.Error())
						continue
					}
					remotelogs.Println("CACHE", "reload policy '"+types.String(policy.Id)+"'")
					storage.UpdatePolicy(policy)
					continue
				}

				remotelogs.Println("CACHE", "restart policy '"+types.String(policy.Id)+"'")

				// 停止老的
				storage.Stop()
				delete(this.storageMap, policy.Id)

				// 启动新的
				storage = this.NewStorageWithPolicy(policy)
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

	return this.policyMap[policyId]
}

// FindStorageWithPolicy 根据策略ID查找存储
func (this *Manager) FindStorageWithPolicy(policyId int64) StorageInterface {
	this.locker.RLock()
	defer this.locker.RUnlock()

	return this.storageMap[policyId]
}

// NewStorageWithPolicy 根据策略获取存储对象
func (this *Manager) NewStorageWithPolicy(policy *serverconfigs.HTTPCachePolicy) StorageInterface {
	switch policy.Type {
	case serverconfigs.CachePolicyStorageFile:
		return NewFileStorage(policy)
	case serverconfigs.CachePolicyStorageMemory:
		return NewMemoryStorage(policy, nil)
	}
	return nil
}

// TotalDiskSize 消耗的磁盘尺寸
func (this *Manager) TotalDiskSize() int64 {
	this.locker.RLock()
	defer this.locker.RUnlock()

	var total = int64(0)
	var sidMap = map[string]bool{} // partition sid => bool
	for _, storage := range this.storageMap {
		// 这里不能直接用 storage.TotalDiskSize() 相加，因为多个缓存策略缓存目录可能处在同一个分区目录下
		fileStorage, ok := storage.(*FileStorage)
		if ok {
			var options = fileStorage.options // copy
			if options != nil {
				var dir = options.Dir // copy
				if len(dir) == 0 {
					continue
				}
				var stat = &unix.Statfs_t{}
				err := unix.Statfs(dir, stat)
				if err != nil {
					continue
				}
				var sid = fmt.Sprintf("%d_%d", stat.Fsid.Val[0], stat.Fsid.Val[1])
				if sidMap[sid] {
					continue
				}
				sidMap[sid] = true
				total += int64(stat.Blocks-stat.Bfree) * int64(stat.Bsize) // we add extra int64() for darwin
			}
		}
	}

	if total < 0 {
		total = 0
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

// FindAllCachePaths 所有缓存路径
func (this *Manager) FindAllCachePaths() []string {
	this.locker.Lock()
	defer this.locker.Unlock()

	var result = []string{}
	for _, policy := range this.policyMap {
		if policy.Type == serverconfigs.CachePolicyStorageFile {
			if policy.Options != nil {
				dir, ok := policy.Options["dir"]
				if ok {
					var dirString = types.String(dir)
					if len(dirString) > 0 {
						result = append(result, dirString)
					}
				}
			}
		}
	}
	return result
}

// FindAllStorages 读取所有缓存存储
func (this *Manager) FindAllStorages() []StorageInterface {
	this.locker.Lock()
	defer this.locker.Unlock()

	var storages = []StorageInterface{}
	for _, storage := range this.storageMap {
		storages = append(storages, storage)
	}
	return storages
}

// ScanGarbageCaches 清理目录中“失联”的缓存文件
func (this *Manager) ScanGarbageCaches(callback func(path string) error) error {
	var storages = this.FindAllStorages()
	for _, storage := range storages {
		fileStorage, ok := storage.(*FileStorage)
		if !ok {
			continue
		}
		err := fileStorage.ScanGarbageCaches(callback)
		if err != nil {
			return err
		}
	}
	return nil
}
