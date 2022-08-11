package iplibrary

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"github.com/TeaOSLab/EdgeCommon/pkg/rpc/pb"
	"github.com/TeaOSLab/EdgeNode/internal/events"
	"github.com/TeaOSLab/EdgeNode/internal/goman"
	"github.com/TeaOSLab/EdgeNode/internal/remotelogs"
	"github.com/TeaOSLab/EdgeNode/internal/rpc"
	"github.com/iwind/TeaGo/Tea"
	_ "github.com/iwind/TeaGo/bootstrap"
	"os"
	"sync"
	"time"
)

var SharedProviderManager = NewProviderManager()

func init() {
	events.On(events.EventLoaded, func() {
		goman.New(func() {
			SharedProviderManager.Start()
		})
	})
	events.On(events.EventQuit, func() {
		SharedProviderManager.Stop()
	})
}

// ProviderManager 中国省份信息管理
type ProviderManager struct {
	ticker *time.Ticker

	cacheFile string

	providerMap map[string]int64 // name => id
	dataHash    string           // 国家JSON的md5

	locker sync.RWMutex

	isUpdated bool
}

func NewProviderManager() *ProviderManager {
	return &ProviderManager{
		cacheFile:   Tea.Root + "/configs/region_provider.json.cache",
		providerMap: map[string]int64{},
	}
}

func (this *ProviderManager) Start() {
	// 从缓存中读取
	err := this.load()
	if err != nil {
		remotelogs.ErrorObject("PROVIDER_MANAGER", err)
	}

	// 第一次更新
	err = this.loop()
	if err != nil {
		remotelogs.ErrorObject("PROVIDER_MANAGER", err)
	}

	// 定时更新
	this.ticker = time.NewTicker(4 * time.Hour)
	for range this.ticker.C {
		err := this.loop()
		if err != nil {
			remotelogs.ErrorObject("PROVIDER_MANAGER", err)
		}
	}
}

func (this *ProviderManager) Stop() {
	if this.ticker != nil {
		this.ticker.Stop()
	}
}

func (this *ProviderManager) Lookup(providerName string) (providerId int64) {
	this.locker.RLock()
	providerId, _ = this.providerMap[providerName]
	this.locker.RUnlock()
	return
}

// 从缓存中读取
func (this *ProviderManager) load() error {
	data, err := os.ReadFile(this.cacheFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	m := map[string]int64{}
	err = json.Unmarshal(data, &m)
	if err != nil {
		return err
	}
	if m != nil && len(m) > 0 {
		this.providerMap = m
	}

	return nil
}

// 更新服务商信息
func (this *ProviderManager) loop() error {
	if this.isUpdated {
		return nil
	}

	rpcClient, err := rpc.SharedRPC()
	if err != nil {
		return err
	}
	resp, err := rpcClient.RegionProviderRPC().FindAllRegionProviders(rpcClient.Context(), &pb.FindAllRegionProvidersRequest{})
	if err != nil {
		return err
	}

	m := map[string]int64{}
	for _, provider := range resp.RegionProviders {
		for _, code := range provider.Codes {
			m[code] = provider.Id
		}
	}

	// 检查是否有更新
	data, err := json.Marshal(m)
	if err != nil {
		return err
	}
	hash := md5.New()
	hash.Write(data)
	dataHash := fmt.Sprintf("%x", hash.Sum(nil))
	if this.dataHash == dataHash {
		return nil
	}
	this.dataHash = dataHash

	this.locker.Lock()
	this.providerMap = m
	this.isUpdated = true
	this.locker.Unlock()

	// 保存到本地缓存

	err = os.WriteFile(this.cacheFile, data, 0666)
	return err
}
