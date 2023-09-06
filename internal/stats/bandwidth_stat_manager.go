// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved. Official site: https://goedge.cn .

package stats

import (
	"encoding/json"
	"github.com/TeaOSLab/EdgeCommon/pkg/nodeconfigs"
	"github.com/TeaOSLab/EdgeCommon/pkg/rpc/pb"
	teaconst "github.com/TeaOSLab/EdgeNode/internal/const"
	"github.com/TeaOSLab/EdgeNode/internal/events"
	"github.com/TeaOSLab/EdgeNode/internal/goman"
	"github.com/TeaOSLab/EdgeNode/internal/remotelogs"
	"github.com/TeaOSLab/EdgeNode/internal/rpc"
	"github.com/TeaOSLab/EdgeNode/internal/utils/fasttime"
	"github.com/iwind/TeaGo/Tea"
	"github.com/iwind/TeaGo/logs"
	"github.com/iwind/TeaGo/types"
	timeutil "github.com/iwind/TeaGo/utils/time"
	"os"
	"sync"
	"time"
)

var SharedBandwidthStatManager = NewBandwidthStatManager()

const bandwidthTimestampDelim = 2 // N秒平均，更为精确

func init() {
	if !teaconst.IsMain {
		return
	}

	events.On(events.EventLoaded, func() {
		goman.New(func() {
			SharedBandwidthStatManager.Start()
		})
	})

	events.OnClose(func() {
		SharedBandwidthStatManager.Cancel()

		err := SharedBandwidthStatManager.Save()
		if err != nil {
			remotelogs.Error("STAT", "save bandwidth stats failed: "+err.Error())
		}
	})
}

type BandwidthStat struct {
	Day      string `json:"day"`
	TimeAt   string `json:"timeAt"`
	UserId   int64  `json:"userId"`
	ServerId int64  `json:"serverId"`

	CurrentBytes     int64 `json:"currentBytes"`
	CurrentTimestamp int64 `json:"currentTimestamp"`
	MaxBytes         int64 `json:"maxBytes"`
	TotalBytes       int64 `json:"totalBytes"`

	CachedBytes         int64 `json:"cachedBytes"`
	AttackBytes         int64 `json:"attackBytes"`
	CountRequests       int64 `json:"countRequests"`
	CountCachedRequests int64 `json:"countCachedRequests"`
	CountAttackRequests int64 `json:"countAttackRequests"`
	UserPlanId          int64 `json:"userPlanId"`
}

// BandwidthStatManager 服务带宽统计
type BandwidthStatManager struct {
	m map[string]*BandwidthStat // serverId@day@time => *BandwidthStat

	pbStats []*pb.ServerBandwidthStat

	lastTime string // 上一次执行的时间

	ticker *time.Ticker
	locker sync.Mutex
}

func NewBandwidthStatManager() *BandwidthStatManager {
	return &BandwidthStatManager{
		m:      map[string]*BandwidthStat{},
		ticker: time.NewTicker(1 * time.Minute), // 时间小于1分钟是为了更快速地上传结果
	}
}

func (this *BandwidthStatManager) Start() {
	// 从上次数据中恢复
	this.locker.Lock()
	this.recover()
	this.locker.Unlock()

	// 循环上报数据
	for range this.ticker.C {
		err := this.Loop()
		if err != nil && !rpc.IsConnError(err) {
			remotelogs.Error("BANDWIDTH_STAT_MANAGER", err.Error())
		}
	}
}

func (this *BandwidthStatManager) Loop() error {
	var regionId int64
	nodeConfig, _ := nodeconfigs.SharedNodeConfig()
	if nodeConfig != nil {
		regionId = nodeConfig.RegionId
	}

	var now = time.Now()
	var day = timeutil.Format("Ymd", now)
	var currentTime = timeutil.FormatTime("Hi", now.Unix()/300*300) // 300s = 5 minutes

	if this.lastTime == currentTime {
		return nil
	}
	this.lastTime = currentTime

	var pbStats = []*pb.ServerBandwidthStat{}

	// 历史未提交记录
	if len(this.pbStats) > 0 {
		var expiredTime = timeutil.FormatTime("Hi", time.Now().Unix()-1200) // 只保留20分钟

		for _, stat := range this.pbStats {
			if stat.TimeAt > expiredTime {
				pbStats = append(pbStats, stat)
			}
		}
		this.pbStats = nil
	}

	this.locker.Lock()
	for key, stat := range this.m {
		if stat.Day < day || stat.TimeAt < currentTime {
			// 防止数据出现错误
			if stat.CachedBytes > stat.TotalBytes || stat.CountCachedRequests == stat.CountRequests {
				stat.CachedBytes = stat.TotalBytes
			}

			if stat.AttackBytes > stat.TotalBytes {
				stat.AttackBytes = stat.TotalBytes
			}

			pbStats = append(pbStats, &pb.ServerBandwidthStat{
				Id:                  0,
				UserId:              stat.UserId,
				ServerId:            stat.ServerId,
				Day:                 stat.Day,
				TimeAt:              stat.TimeAt,
				Bytes:               stat.MaxBytes / bandwidthTimestampDelim,
				TotalBytes:          stat.TotalBytes,
				CachedBytes:         stat.CachedBytes,
				AttackBytes:         stat.AttackBytes,
				CountRequests:       stat.CountRequests,
				CountCachedRequests: stat.CountCachedRequests,
				CountAttackRequests: stat.CountAttackRequests,
				UserPlanId:          stat.UserPlanId,
				NodeRegionId:        regionId,
			})
			delete(this.m, key)
		}
	}
	this.locker.Unlock()

	if len(pbStats) > 0 {
		// 上传
		rpcClient, err := rpc.SharedRPC()
		if err != nil {
			return err
		}
		_, err = rpcClient.ServerBandwidthStatRPC.UploadServerBandwidthStats(rpcClient.Context(), &pb.UploadServerBandwidthStatsRequest{ServerBandwidthStats: pbStats})
		if err != nil {
			this.pbStats = pbStats

			return err
		}
	}

	return nil
}

// AddBandwidth 添加带宽数据
func (this *BandwidthStatManager) AddBandwidth(userId int64, userPlanId int64, serverId int64, peekBytes int64, totalBytes int64) {
	if serverId <= 0 || (peekBytes == 0 && totalBytes == 0) {
		return
	}

	var now = fasttime.Now()
	var timestamp = now.Unix() / bandwidthTimestampDelim * bandwidthTimestampDelim // 将时间戳均分成N等份
	var day = now.Ymd()
	var timeAt = now.Round5Hi()
	var key = types.String(serverId) + "@" + day + "@" + timeAt

	// 增加TCP Header尺寸，这里默认MTU为1500，且默认为IPv4
	const mtu = 1500
	const tcpHeaderSize = 20
	if peekBytes > mtu {
		peekBytes += peekBytes * tcpHeaderSize / mtu
	}

	this.locker.Lock()
	stat, ok := this.m[key]
	if ok {
		// 此刻如果发生用户ID（userId）的变化也忽略，等N分钟后有新记录后再换

		if stat.CurrentTimestamp == timestamp {
			stat.CurrentBytes += peekBytes
		} else {
			stat.CurrentBytes = peekBytes
			stat.CurrentTimestamp = timestamp
		}
		if stat.CurrentBytes > stat.MaxBytes {
			stat.MaxBytes = stat.CurrentBytes
		}

		stat.TotalBytes += totalBytes
	} else {
		this.m[key] = &BandwidthStat{
			Day:              day,
			TimeAt:           timeAt,
			UserId:           userId,
			UserPlanId:       userPlanId,
			ServerId:         serverId,
			CurrentBytes:     peekBytes,
			MaxBytes:         peekBytes,
			TotalBytes:       totalBytes,
			CurrentTimestamp: timestamp,
		}
	}
	this.locker.Unlock()
}

// AddTraffic 添加请求数据
func (this *BandwidthStatManager) AddTraffic(serverId int64, cachedBytes int64, countRequests int64, countCachedRequests int64, countAttacks int64, attackBytes int64) {
	var now = fasttime.Now()
	var day = now.Ymd()
	var timeAt = now.Round5Hi()
	var key = types.String(serverId) + "@" + day + "@" + timeAt
	this.locker.Lock()
	// 只有有记录了才会添加
	stat, ok := this.m[key]
	if ok {
		stat.CachedBytes += cachedBytes
		stat.CountRequests += countRequests
		stat.CountCachedRequests += countCachedRequests
		stat.CountAttackRequests += countAttacks
		stat.AttackBytes += attackBytes
	}
	this.locker.Unlock()
}

func (this *BandwidthStatManager) Inspect() {
	this.locker.Lock()
	logs.PrintAsJSON(this.m)
	this.locker.Unlock()
}

func (this *BandwidthStatManager) Map() map[int64]int64 /** serverId => max bytes **/ {
	this.locker.Lock()
	defer this.locker.Unlock()

	var m = map[int64]int64{}
	for _, v := range this.m {
		m[v.ServerId] = v.MaxBytes / bandwidthTimestampDelim
	}

	return m
}

// Save 保存到本地磁盘
func (this *BandwidthStatManager) Save() error {
	this.locker.Lock()
	defer this.locker.Unlock()

	if len(this.m) == 0 {
		return nil
	}

	data, err := json.Marshal(this.m)
	if err != nil {
		return err
	}

	_ = os.Remove(this.cacheFile())
	return os.WriteFile(this.cacheFile(), data, 0666)
}

// Cancel 取消上传
func (this *BandwidthStatManager) Cancel() {
	this.ticker.Stop()
}

// 从本地缓存文件中恢复数据
func (this *BandwidthStatManager) recover() {
	cacheData, err := os.ReadFile(this.cacheFile())
	if err == nil {
		var m = map[string]*BandwidthStat{}
		err = json.Unmarshal(cacheData, &m)
		if err == nil && len(m) > 0 {
			var lastTime = ""
			for _, stat := range m {
				if stat.Day != fasttime.Now().Ymd() {
					continue
				}
				if len(lastTime) == 0 || stat.TimeAt > lastTime {
					lastTime = stat.TimeAt
				}
			}
			if len(lastTime) > 0 {
				var availableTime = timeutil.FormatTime("Hi", (time.Now().Unix()-300) /** 只保留5分钟的 **/ /300*300) // 300s = 5 minutes
				if lastTime >= availableTime {
					this.m = m
					this.lastTime = lastTime
				}
			}
		}
		_ = os.Remove(this.cacheFile())
	}
}

// 获取缓存文件
// 不能在init()中初始化，避免无法获得正确的路径
func (this *BandwidthStatManager) cacheFile() string {
	return Tea.Root + "/data/bandwidth.dat"
}
