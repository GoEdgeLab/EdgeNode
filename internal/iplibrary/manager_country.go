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

var SharedCountryManager = NewCountryManager()

func init() {
	events.On(events.EventLoaded, func() {
		goman.New(func() {
			SharedCountryManager.Start()
		})
	})
	events.On(events.EventQuit, func() {
		SharedCountryManager.Stop()
	})
}

// CountryManager 国家/地区信息管理
type CountryManager struct {
	ticker *time.Ticker

	cacheFile string

	countryMap map[string]int64 // countryName => countryId
	dataHash   string           // 国家JSON的md5

	locker sync.RWMutex

	isUpdated bool
}

func NewCountryManager() *CountryManager {
	return &CountryManager{
		cacheFile:  Tea.Root + "/configs/region_country.json.cache",
		countryMap: map[string]int64{},
	}
}

func (this *CountryManager) Start() {
	// 从缓存中读取
	err := this.load()
	if err != nil {
		remotelogs.ErrorObject("COUNTRY_MANAGER", err)
	}

	// 第一次更新
	err = this.loop()
	if err != nil {
		remotelogs.ErrorObject("COUNTRY_MANAGER", err)
	}

	// 定时更新
	this.ticker = time.NewTicker(4 * time.Hour)
	for range this.ticker.C {
		err := this.loop()
		if err != nil {
			remotelogs.ErrorObject("COUNTRY_MANAGER", err)
		}
	}
}

func (this *CountryManager) Stop() {
	if this.ticker != nil {
		this.ticker.Stop()
	}
}

func (this *CountryManager) Lookup(countryName string) (countryId int64) {
	this.locker.RLock()
	countryId, _ = this.countryMap[countryName]
	this.locker.RUnlock()
	return countryId
}

// 从缓存中读取
func (this *CountryManager) load() error {
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
		this.countryMap = m
	}

	return nil
}

// 更新国家信息
func (this *CountryManager) loop() error {
	if this.isUpdated {
		return nil
	}

	rpcClient, err := rpc.SharedRPC()
	if err != nil {
		return err
	}
	resp, err := rpcClient.RegionCountryRPC().FindAllRegionCountries(rpcClient.Context(), &pb.FindAllRegionCountriesRequest{})
	if err != nil {
		return err
	}

	m := map[string]int64{}
	for _, country := range resp.RegionCountries {
		for _, code := range country.Codes {
			m[code] = country.Id
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
	this.countryMap = m
	this.isUpdated = true
	this.locker.Unlock()

	// 保存到本地缓存
	err = os.WriteFile(this.cacheFile, data, 0666)
	return err
}
