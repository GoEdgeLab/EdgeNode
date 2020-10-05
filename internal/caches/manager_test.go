package caches

import (
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs"
	"github.com/iwind/TeaGo/Tea"
	"testing"
)

func TestManager_UpdatePolicies(t *testing.T) {
	{
		policies := []*serverconfigs.HTTPCachePolicy{}
		SharedManager.UpdatePolicies(policies)
		printManager(t)
	}

	{
		policies := []*serverconfigs.HTTPCachePolicy{
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
		policies := []*serverconfigs.HTTPCachePolicy{
			{
				Id:   1,
				Type: serverconfigs.CachePolicyStorageFile,
				Options: map[string]interface{}{
					"dir": Tea.Root + "/caches",
				},
			},
			{
				Id:      2,
				Type:    serverconfigs.CachePolicyStorageFile,
				MaxKeys: 1,
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

func printManager(t *testing.T) {
	t.Log("===manager==")
	t.Log("storage:")
	for _, storage := range SharedManager.storageMap {
		t.Log("  storage:", storage.Policy().Id)
	}
	t.Log("===============")
}
