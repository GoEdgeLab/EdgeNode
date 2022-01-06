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
	"github.com/TeaOSLab/EdgeNode/internal/utils"
	"github.com/iwind/TeaGo/Tea"
	_ "github.com/iwind/TeaGo/bootstrap"
	"io/ioutil"
	"os"
	"sync"
	"time"
)

const (
	ChinaCountryId int64 = 1
)

var SharedProvinceManager = NewProvinceManager()

func init() {
	events.On(events.EventLoaded, func() {
		goman.New(func() {
			SharedProvinceManager.Start()
		})
	})
}

// ProvinceManager 中国省份信息管理
type ProvinceManager struct {
	cacheFile string

	provinceMap map[string]int64 // provinceName => provinceId
	dataHash    string           // 国家JSON的md5

	locker sync.RWMutex

	isUpdated bool
}

func NewProvinceManager() *ProvinceManager {
	return &ProvinceManager{
		cacheFile:   Tea.Root + "/configs/region_province.json.cache",
		provinceMap: map[string]int64{},
	}
}

func (this *ProvinceManager) Start() {
	// 从缓存中读取
	err := this.load()
	if err != nil {
		remotelogs.ErrorObject("PROVINCE_MANAGER", err)
	}

	// 第一次更新
	err = this.loop()
	if err != nil {
		remotelogs.ErrorObject("PROVINCE_MANAGER", err)
	}

	// 定时更新
	ticker := utils.NewTicker(4 * time.Hour)
	events.On(events.EventQuit, func() {
		ticker.Stop()
	})
	for ticker.Next() {
		err := this.loop()
		if err != nil {
			remotelogs.ErrorObject("PROVINCE_MANAGER", err)
		}
	}
}

func (this *ProvinceManager) Lookup(provinceName string) (provinceId int64) {
	this.locker.RLock()
	provinceId, _ = this.provinceMap[provinceName]
	this.locker.RUnlock()
	return provinceId
}

// 从缓存中读取
func (this *ProvinceManager) load() error {
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
		this.provinceMap = m
	}

	return nil
}

// 更新省份信息
func (this *ProvinceManager) loop() error {
	if this.isUpdated {
		return nil
	}

	rpcClient, err := rpc.SharedRPC()
	if err != nil {
		return err
	}
	resp, err := rpcClient.RegionProvinceRPC().FindAllEnabledRegionProvincesWithCountryId(rpcClient.Context(), &pb.FindAllEnabledRegionProvincesWithCountryIdRequest{
		RegionCountryId: ChinaCountryId,
	})
	if err != nil {
		return err
	}

	m := map[string]int64{}
	for _, province := range resp.RegionProvinces {
		for _, code := range province.Codes {
			m[code] = province.Id
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
	this.provinceMap = m
	this.isUpdated = true
	this.locker.Unlock()

	// 保存到本地缓存

	err = ioutil.WriteFile(this.cacheFile, data, 0666)
	return err
}
