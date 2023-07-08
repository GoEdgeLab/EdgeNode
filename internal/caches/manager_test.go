package caches

import (
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs"
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs/shared"
	"github.com/iwind/TeaGo/Tea"
	"testing"
)

func TestManager_UpdatePolicies(t *testing.T) {
	{
		var policies = []*serverconfigs.HTTPCachePolicy{}
		SharedManager.UpdatePolicies(policies)
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
		SharedManager.UpdatePolicies(policies)
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
		SharedManager.UpdatePolicies(policies)
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
	SharedManager.UpdatePolicies(policies)
	SharedManager.UpdatePolicies([]*serverconfigs.HTTPCachePolicy{
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
	SharedManager.UpdatePolicies(policies)
	SharedManager.UpdatePolicies([]*serverconfigs.HTTPCachePolicy{
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

func printManager(t *testing.T) {
	t.Log("===manager==")
	t.Log("storage:")
	for _, storage := range SharedManager.storageMap {
		t.Log("  storage:", storage.Policy().Id)
	}
	t.Log("===============")
}
