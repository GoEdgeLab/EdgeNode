// Copyright 2024 GoEdge CDN goedge.cdn@gmail.com. All rights reserved. Official site: https://goedge.cn .

package stats

import (
	"encoding/json"
	teaconst "github.com/TeaOSLab/EdgeNode/internal/const"
	"github.com/TeaOSLab/EdgeNode/internal/events"
	"github.com/TeaOSLab/EdgeNode/internal/goman"
	"github.com/TeaOSLab/EdgeNode/internal/remotelogs"
	"github.com/TeaOSLab/EdgeNode/internal/trackers"
	"github.com/TeaOSLab/EdgeNode/internal/utils/fasttime"
	fsutils "github.com/TeaOSLab/EdgeNode/internal/utils/fs"
	"github.com/TeaOSLab/EdgeNode/internal/utils/kvstore"
	"github.com/iwind/TeaGo/Tea"
	"github.com/iwind/TeaGo/types"
	timeutil "github.com/iwind/TeaGo/utils/time"
	"os"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

var SharedDAUManager = NewDAUManager()

func init() {
	if teaconst.IsMain {
		err := SharedDAUManager.Init()
		if err != nil {
			remotelogs.Error("DAU_MANAGER", "initialize DAU manager failed: "+err.Error())
		}
	}
}

type IPInfo struct {
	IP       string
	ServerId int64
}

type DAUManager struct {
	cacheFile string

	ipChan  chan IPInfo
	ipTable *kvstore.Table[[]byte] // server_DATE_serverId_ip => nil

	statMap    map[string]int64 // server_DATE_serverId => count
	statLocker sync.RWMutex

	cleanTicker *time.Ticker
}

// NewDAUManager DAU计算器
func NewDAUManager() *DAUManager {
	return &DAUManager{
		cacheFile:   Tea.Root + "/data/stat_dau.cache",
		statMap:     map[string]int64{},
		cleanTicker: time.NewTicker(24 * time.Hour),
		ipChan:      make(chan IPInfo, 8192),
	}
}

func (this *DAUManager) Init() error {
	// recover from cache
	_ = this.recover()

	// create table
	store, storeErr := kvstore.DefaultStore()
	if storeErr != nil {
		return storeErr
	}

	db, dbErr := store.NewDB("dau")
	if dbErr != nil {
		return dbErr
	}

	{
		table, err := kvstore.NewTable[[]byte]("ip", kvstore.NewNilValueEncoder())
		if err != nil {
			return err
		}
		db.AddTable(table)
		this.ipTable = table
	}

	{
		table, err := kvstore.NewTable[uint64]("stats", kvstore.NewIntValueEncoder[uint64]())
		if err != nil {
			return err
		}
		db.AddTable(table)
	}

	// clean expires items
	goman.New(func() {
		for range this.cleanTicker.C {
			fsutils.WaitLoad(15, 16, 1*time.Hour)

			err := this.CleanStats()
			if err != nil {
				remotelogs.Error("DAU_MANAGER", "clean stats failed: "+err.Error())
			}
		}
	})

	// dump ip to kvstore
	goman.New(func() {
		// cache latest IPs to reduce kv queries
		var cachedIPs []IPInfo
		var maxIPs = runtime.NumCPU() * 8
		if maxIPs <= 0 {
			maxIPs = 8
		} else if maxIPs > 64 {
			maxIPs = 64
		}

		var day = fasttime.Now().Ymd()

	Loop:
		for ipInfo := range this.ipChan {
			// check day
			if fasttime.Now().Ymd() != day {
				day = fasttime.Now().Ymd()
				cachedIPs = []IPInfo{}
			}

			// lookup cache
			for _, cachedIP := range cachedIPs {
				if cachedIP.IP == ipInfo.IP && cachedIP.ServerId == ipInfo.ServerId {
					continue Loop
				}
			}

			// add to cache
			cachedIPs = append(cachedIPs, ipInfo)
			if len(cachedIPs) > maxIPs {
				cachedIPs = cachedIPs[1:]
			}

			_ = this.processIP(ipInfo.ServerId, ipInfo.IP)
		}
	})

	// dump to cache when close
	events.OnClose(func() {
		_ = this.Close()
	})

	return nil
}

func (this *DAUManager) AddIP(serverId int64, ip string) {
	select {
	case this.ipChan <- IPInfo{
		IP:       ip,
		ServerId: serverId,
	}:
	default:
	}
}

func (this *DAUManager) processIP(serverId int64, ip string) error {
	// day
	var date = fasttime.Now().Ymd()

	{
		var key = "server_" + date + "_" + types.String(serverId) + "_" + ip
		found, err := this.ipTable.Exist(key)
		if err != nil || found {
			return err
		}

		err = this.ipTable.Set(key, nil)
		if err != nil {
			return err
		}
	}

	{
		var key = "server_" + date + "_" + types.String(serverId)
		this.statLocker.Lock()
		this.statMap[key] = this.statMap[key] + 1
		this.statLocker.Unlock()
	}

	return nil
}

func (this *DAUManager) ReadStatMap() map[string]int64 {
	this.statLocker.Lock()
	var statMap = this.statMap
	this.statMap = map[string]int64{}
	this.statLocker.Unlock()
	return statMap
}

func (this *DAUManager) Flush() error {
	return this.ipTable.DB().Store().Flush()
}

func (this *DAUManager) TestInspect(t *testing.T) {
	err := this.ipTable.DB().Inspect(func(key []byte, value []byte) {
		t.Log(string(key), "=>", string(value))
	})
	if err != nil {
		t.Fatal(err)
	}
}

func (this *DAUManager) Close() error {
	this.statLocker.Lock()
	var statMap = this.statMap
	this.statMap = map[string]int64{}
	this.statLocker.Unlock()

	if len(statMap) == 0 {
		return nil
	}

	statJSON, err := json.Marshal(statMap)
	if err != nil {
		return err
	}

	return os.WriteFile(this.cacheFile, statJSON, 0666)
}

func (this *DAUManager) CleanStats() error {
	var tr = trackers.Begin("STAT:DAU_CLEAN_STATS")
	defer tr.End()

	// day
	{
		var date = timeutil.Format("Ymd", time.Now().AddDate(0, 0, -2))
		err := this.ipTable.DeleteRange("server_", "server_"+date)
		if err != nil {
			return err
		}
	}

	return nil
}

func (this *DAUManager) Truncate() error {
	return this.ipTable.Truncate()
}

func (this *DAUManager) recover() error {
	data, err := os.ReadFile(this.cacheFile)
	if err != nil || len(data) == 0 {
		return err
	}

	_ = os.Remove(this.cacheFile)

	var statMap = map[string]int64{}
	err = json.Unmarshal(data, &statMap)
	if err != nil {
		return err
	}

	var today = timeutil.Format("Ymd")
	for key := range statMap {
		var pieces = strings.Split(key, "_")
		if pieces[1] != today {
			delete(statMap, key)
		}
	}
	this.statMap = statMap
	return nil
}
