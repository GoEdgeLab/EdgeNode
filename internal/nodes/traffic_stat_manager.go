package nodes

import (
	"github.com/TeaOSLab/EdgeCommon/pkg/rpc/pb"
	"github.com/TeaOSLab/EdgeNode/internal/logs"
	"github.com/TeaOSLab/EdgeNode/internal/rpc"
	"github.com/TeaOSLab/EdgeNode/internal/utils"
	"github.com/iwind/TeaGo/Tea"
	"strconv"
	"sync"
	"time"
)

var SharedTrafficStatManager = NewTrafficStatManager()

// 流量统计
type TrafficStatManager struct {
	m      map[string]int64 // [timestamp serverId] => bytes
	locker sync.Mutex
}

// 获取新对象
func NewTrafficStatManager() *TrafficStatManager {
	manager := &TrafficStatManager{
		m: map[string]int64{},
	}

	go manager.Start()

	return manager
}

// 启动自动任务
func (this *TrafficStatManager) Start() {
	duration := 5 * time.Minute
	if Tea.IsTesting() {
		// 测试环境缩短上传时间，方便我们调试
		duration = 30 * time.Second
	}
	ticker := time.NewTicker(duration)
	for range ticker.C {
		err := this.Upload()
		if err != nil {
			logs.Error("TRAFFIC_STAT_MANAGER", "upload stats failed: "+err.Error())
		}
	}
}

// 添加流量
func (this *TrafficStatManager) Add(serverId int64, bytes int64) {
	if bytes == 0 {
		return
	}

	timestamp := utils.UnixTime() / 300 * 300

	key := strconv.FormatInt(timestamp, 10) + strconv.FormatInt(serverId, 10)
	this.locker.Lock()
	this.m[key] += bytes
	this.locker.Unlock()
}

// 上传流量
func (this *TrafficStatManager) Upload() error {
	if sharedNodeConfig == nil {
		return nil
	}

	client, err := rpc.SharedRPC()
	if err != nil {
		return err
	}

	this.locker.Lock()
	m := this.m
	this.m = map[string]int64{}
	this.locker.Unlock()

	pbStats := []*pb.ServerDailyStat{}
	for key, bytes := range m {
		timestamp, err := strconv.ParseInt(key[:10], 10, 64)
		if err != nil {
			return err
		}
		serverId, err := strconv.ParseInt(key[10:], 10, 64)
		if err != nil {
			return err
		}

		pbStats = append(pbStats, &pb.ServerDailyStat{
			ServerId:  serverId,
			RegionId:  sharedNodeConfig.RegionId,
			Bytes:     bytes,
			CreatedAt: timestamp,
		})
	}
	if len(pbStats) == 0 {
		return nil
	}
	_, err = client.ServerDailyStatRPC().UploadServerDailyStats(client.Context(), &pb.UploadServerDailyStatsRequest{Stats: pbStats})
	return err
}
