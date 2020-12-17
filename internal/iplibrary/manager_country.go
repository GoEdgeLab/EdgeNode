package iplibrary

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"github.com/TeaOSLab/EdgeCommon/pkg/rpc/pb"
	"github.com/TeaOSLab/EdgeNode/internal/events"
	"github.com/TeaOSLab/EdgeNode/internal/remotelogs"
	"github.com/TeaOSLab/EdgeNode/internal/rpc"
	"github.com/TeaOSLab/EdgeNode/internal/utils"
	"github.com/iwind/TeaGo/Tea"
	_ "github.com/iwind/TeaGo/bootstrap"
	"io/ioutil"
	"os"
	"sync"
	"time"
)

var SharedCountryManager = NewCountryManager()

func init() {
	events.On(events.EventStart, func() {
		go SharedCountryManager.Start()
	})
}

// 国家信息管理
type CountryManager struct {
	cacheFile string

	countryMap map[string]int64 // countryName => countryId
	dataHash   string           // 国家JSON的md5

	locker sync.RWMutex
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
		remotelogs.Error("COUNTRY_MANAGER", err.Error())
	}

	// 第一次更新
	err = this.loop()
	if err != nil {
		remotelogs.Error("COUNTRY_MANAGER", err.Error())
	}

	// 定时更新
	ticker := utils.NewTicker(1 * time.Hour)
	events.On(events.EventQuit, func() {
		ticker.Stop()
	})
	for range ticker.C {
		err := this.loop()
		if err != nil {
			remotelogs.Error("COUNTRY_MANAGER", err.Error())
		}
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
	data, err := ioutil.ReadFile(this.cacheFile)
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
	rpcClient, err := rpc.SharedRPC()
	if err != nil {
		return err
	}
	resp, err := rpcClient.RegionCountryRPC().FindAllEnabledRegionCountries(rpcClient.Context(), &pb.FindAllEnabledRegionCountriesRequest{})
	if err != nil {
		return err
	}

	m := map[string]int64{}
	for _, country := range resp.Countries {
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
	this.locker.Unlock()

	// 保存到本地缓存
	err = ioutil.WriteFile(this.cacheFile, data, 0666)
	return err
}
