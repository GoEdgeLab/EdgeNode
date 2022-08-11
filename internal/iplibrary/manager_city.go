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
	"github.com/iwind/TeaGo/types"
	"os"
	"sync"
	"time"
)

var SharedCityManager = NewCityManager()

func init() {
	events.On(events.EventLoaded, func() {
		goman.New(func() {
			SharedCityManager.Start()
		})
	})
	events.On(events.EventQuit, func() {
		SharedCityManager.Stop()
	})
}

// CityManager 中国省份信息管理
type CityManager struct {
	ticker *time.Ticker

	cacheFile string

	cityMap  map[string]int64 // provinceName_cityName => cityName
	dataHash string           // 国家JSON的md5

	locker sync.RWMutex

	isUpdated bool
}

func NewCityManager() *CityManager {
	return &CityManager{
		cacheFile: Tea.Root + "/configs/region_city.json.cache",
		cityMap:   map[string]int64{},
	}
}

func (this *CityManager) Start() {
	// 从缓存中读取
	err := this.load()
	if err != nil {
		remotelogs.ErrorObject("CITY_MANAGER", err)
	}

	// 第一次更新
	err = this.loop()
	if err != nil {
		remotelogs.ErrorObject("City_MANAGER", err)
	}

	// 定时更新
	this.ticker = time.NewTicker(4 * time.Hour)
	for range this.ticker.C {
		err := this.loop()
		if err != nil {
			remotelogs.ErrorObject("CITY_MANAGER", err)
		}
	}
}

func (this *CityManager) Stop() {
	if this.ticker != nil {
		this.ticker.Stop()
	}
}

func (this *CityManager) Lookup(provinceId int64, cityName string) (cityId int64) {
	this.locker.RLock()
	cityId, _ = this.cityMap[types.String(provinceId)+"_"+cityName]
	this.locker.RUnlock()
	return
}

// 从缓存中读取
func (this *CityManager) load() error {
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
		this.cityMap = m
	}

	return nil
}

// 更新城市信息
func (this *CityManager) loop() error {
	if this.isUpdated {
		return nil
	}

	rpcClient, err := rpc.SharedRPC()
	if err != nil {
		return err
	}
	resp, err := rpcClient.RegionCityRPC().FindAllRegionCities(rpcClient.Context(), &pb.FindAllRegionCitiesRequest{})
	if err != nil {
		return err
	}

	m := map[string]int64{}
	for _, city := range resp.RegionCities {
		for _, code := range city.Codes {
			m[types.String(city.RegionProvinceId)+"_"+code] = city.Id
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
	this.cityMap = m
	this.isUpdated = true
	this.locker.Unlock()

	// 保存到本地缓存

	err = os.WriteFile(this.cacheFile, data, 0666)
	return err
}
