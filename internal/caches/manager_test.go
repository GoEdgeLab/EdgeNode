package caches_test

import (
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs"
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs/shared"
	"github.com/TeaOSLab/EdgeNode/internal/caches"
	"github.com/iwind/TeaGo/Tea"
	"testing"
)

func TestManager_UpdatePolicies(t *testing.T) {
	{
		var policies = []*serverconfigs.HTTPCachePolicy{}
		caches.SharedManager.UpdatePolicies(policies)
		printManager(t)
	}

	{
		var policies = []*serverconfigs.HTTPCachePolicy{
			{
				Id:   1,
				Type: serverconfigs.CachePolicyStorageFile,
				Options: map[string]interface{}{
					"dir": Tea.Root + "/caches",
				},
			},
			{
				Id:   2,
				Type: serverconfigs.CachePolicyStorageFile,
				Options: map[string]interface{}{
					"dir": Tea.Root + "/caches",
				},
			},
			{
				Id:   3,
				Type: serverconfigs.CachePolicyStorageFile,
				Options: map[string]interface{}{
					"dir": Tea.Root + "/caches",
				},
			},
		}
		caches.SharedManager.UpdatePolicies(policies)
		printManager(t)
	}

	{
		var policies = []*serverconfigs.HTTPCachePolicy{
			{
				Id:   1,
				Type: serverconfigs.CachePolicyStorageFile,
				Options: map[string]interface{}{
					"dir": Tea.Root + "/caches",
				},
			},
			{
				Id:   2,
				Type: serverconfigs.CachePolicyStorageFile,
				Options: map[string]interface{}{
					"dir": Tea.Root + "/caches",
				},
			},
			{
				Id:   4,
				Type: serverconfigs.CachePolicyStorageFile,
				Options: map[string]interface{}{
					"dir": Tea.Root + "/caches",
				},
			},
		}
		caches.SharedManager.UpdatePolicies(policies)
		printManager(t)
	}
}

func TestManager_ChangePolicy_Memory(t *testing.T) {
	var policies = []*serverconfigs.HTTPCachePolicy{
		{
			Id:       1,
			Type:     serverconfigs.CachePolicyStorageMemory,
			Options:  map[string]interface{}{},
			Capacity: &shared.SizeCapacity{Count: 1, Unit: shared.SizeCapacityUnitGB},
		},
	}
	caches.SharedManager.UpdatePolicies(policies)
	caches.SharedManager.UpdatePolicies([]*serverconfigs.HTTPCachePolicy{
		{
			Id:       1,
			Type:     serverconfigs.CachePolicyStorageMemory,
			Options:  map[string]interface{}{},
			Capacity: &shared.SizeCapacity{Count: 2, Unit: shared.SizeCapacityUnitGB},
		},
	})
}

func TestManager_ChangePolicy_File(t *testing.T) {
	var policies = []*serverconfigs.HTTPCachePolicy{
		{
			Id:   1,
			Type: serverconfigs.CachePolicyStorageFile,
			Options: map[string]interface{}{
				"dir": Tea.Root + "/data/cache-index/p1",
			},
			Capacity: &shared.SizeCapacity{Count: 1, Unit: shared.SizeCapacityUnitGB},
		},
	}
	caches.SharedManager.UpdatePolicies(policies)
	caches.SharedManager.UpdatePolicies([]*serverconfigs.HTTPCachePolicy{
		{
			Id:   1,
			Type: serverconfigs.CachePolicyStorageFile,
			Options: map[string]interface{}{
				"dir": Tea.Root + "/data/cache-index/p1",
			},
			Capacity: &shared.SizeCapacity{Count: 2, Unit: shared.SizeCapacityUnitGB},
		},
	})
}

func TestManager_MaxSystemMemoryBytesPerStorage(t *testing.T) {
	for i := 0; i < 100; i++ {
		caches.SharedManager.CountMemoryStorages = i
		t.Log(i, caches.SharedManager.MaxSystemMemoryBytesPerStorage()>>30, "GB")
	}
}

func printManager(t *testing.T) {
	t.Log("===manager==")
	t.Log("storage:")
	for _, storage := range caches.SharedManager.StorageMap() {
		t.Log("  storage:", storage.Policy().Id)
	}
	t.Log("===============")
}
