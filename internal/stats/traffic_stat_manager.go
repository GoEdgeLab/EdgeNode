package stats

import (
	"github.com/TeaOSLab/EdgeCommon/pkg/nodeconfigs"
	"github.com/TeaOSLab/EdgeCommon/pkg/rpc/pb"
	"github.com/TeaOSLab/EdgeNode/internal/events"
	"github.com/TeaOSLab/EdgeNode/internal/remotelogs"
	"github.com/TeaOSLab/EdgeNode/internal/rpc"
	"github.com/TeaOSLab/EdgeNode/internal/utils"
	"github.com/iwind/TeaGo/Tea"
	"strconv"
	"sync"
	"time"
)

var SharedTrafficStatManager = NewTrafficStatManager()

type TrafficItem struct {
	Bytes               int64
	CachedBytes         int64
	CountRequests       int64
	CountCachedRequests int64
}

// TrafficStatManager 区域流量统计
type TrafficStatManager struct {
	itemMap    map[string]*TrafficItem // [timestamp serverId] => bytes
	locker     sync.Mutex
	configFunc func() *nodeconfigs.NodeConfig
}

// NewTrafficStatManager 获取新对象
func NewTrafficStatManager() *TrafficStatManager {
	manager := &TrafficStatManager{
		itemMap: map[string]*TrafficItem{},
	}

	return manager
}

// Start 启动自动任务
func (this *TrafficStatManager) Start(configFunc func() *nodeconfigs.NodeConfig) {
	this.configFunc = configFunc

	duration := 5 * time.Minute
	if Tea.IsTesting() {
		// 测试环境缩短上传时间，方便我们调试
		duration = 30 * time.Second
	}
	ticker := time.NewTicker(duration)
	events.On(events.EventQuit, func() {
		remotelogs.Println("TRAFFIC_STAT_MANAGER", "quit")
		ticker.Stop()
	})
	remotelogs.Println("TRAFFIC_STA_MANAGER", "start ...")
	for range ticker.C {
		err := this.Upload()
		if err != nil {
			remotelogs.Error("TRAFFIC_STAT_MANAGER", "upload stats failed: "+err.Error())
		}
	}
}

// Add 添加流量
func (this *TrafficStatManager) Add(serverId int64, bytes int64, cachedBytes int64, countRequests int64, countCachedRequests int64) {
	if bytes == 0 {
		return
	}

	timestamp := utils.UnixTime() / 300 * 300

	key := strconv.FormatInt(timestamp, 10) + strconv.FormatInt(serverId, 10)
	this.locker.Lock()
	item, ok := this.itemMap[key]
	if !ok {
		item = &TrafficItem{}
		this.itemMap[key] = item
	}
	item.Bytes += bytes
	item.CachedBytes += cachedBytes
	item.CountRequests += countRequests
	item.CountCachedRequests += countCachedRequests
	this.locker.Unlock()
}

// Upload 上传流量
func (this *TrafficStatManager) Upload() error {
	config := this.configFunc()
	if config == nil {
		return nil
	}

	client, err := rpc.SharedRPC()
	if err != nil {
		return err
	}

	this.locker.Lock()
	m := this.itemMap
	this.itemMap = map[string]*TrafficItem{}
	this.locker.Unlock()

	pbStats := []*pb.ServerDailyStat{}
	for key, item := range m {
		timestamp, err := strconv.ParseInt(key[:10], 10, 64)
		if err != nil {
			return err
		}
		serverId, err := strconv.ParseInt(key[10:], 10, 64)
		if err != nil {
			return err
		}

		pbStats = append(pbStats, &pb.ServerDailyStat{
			ServerId:            serverId,
			RegionId:            config.RegionId,
			Bytes:               item.Bytes,
			CachedBytes:         item.CachedBytes,
			CountRequests:       item.CountRequests,
			CountCachedRequests: item.CountCachedRequests,
			CreatedAt:           timestamp,
		})
	}
	if len(pbStats) == 0 {
		return nil
	}
	_, err = client.ServerDailyStatRPC().UploadServerDailyStats(client.Context(), &pb.UploadServerDailyStatsRequest{Stats: pbStats})
	return err
}
