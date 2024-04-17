// Copyright 2024 GoEdge CDN goedge.cdn@gmail.com. All rights reserved. Official site: https://goedge.cn .

package metrics

import (
	"github.com/TeaOSLab/EdgeCommon/pkg/rpc/pb"
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs"
	"github.com/TeaOSLab/EdgeNode/internal/goman"
	"github.com/TeaOSLab/EdgeNode/internal/remotelogs"
	"github.com/TeaOSLab/EdgeNode/internal/rpc"
	"github.com/TeaOSLab/EdgeNode/internal/trackers"
	"github.com/TeaOSLab/EdgeNode/internal/utils"
	byteutils "github.com/TeaOSLab/EdgeNode/internal/utils/byte"
	"github.com/TeaOSLab/EdgeNode/internal/utils/idles"
	"github.com/TeaOSLab/EdgeNode/internal/utils/kvstore"
	"github.com/TeaOSLab/EdgeNode/internal/zero"
	"github.com/cockroachdb/pebble"
	"github.com/iwind/TeaGo/Tea"
	"github.com/iwind/TeaGo/types"
	"strings"
	"sync"
	"testing"
	"time"
)

// TODO sumValues不用每次insertStat的时候都保存

// KVTask KV存储实现的任务管理
type KVTask struct {
	BaseTask

	itemsTable  *kvstore.Table[*Stat]  // hash => *Stat
	valuesTable *kvstore.Table[[]byte] // time_version_serverId_value_hash => []byte(nil)
	sumTable    *kvstore.Table[[]byte] // time_version_serverId => [count][total]

	serverTimeMap     map[string]zero.Zero // 有变更的网站 serverId_time => Zero
	serverIdMapLocker sync.Mutex

	statsTicker  *utils.Ticker
	cleanTicker  *time.Ticker
	uploadTicker *utils.Ticker

	valuesCacheMap map[string]int64 // hash => value
}

func NewKVTask(itemConfig *serverconfigs.MetricItemConfig) *KVTask {
	return &KVTask{
		BaseTask: BaseTask{
			itemConfig: itemConfig,
			statsMap:   map[string]*Stat{},
		},

		serverTimeMap:  map[string]zero.Zero{},
		valuesCacheMap: map[string]int64{},
	}
}

func (this *KVTask) Init() error {
	store, err := kvstore.DefaultStore()
	if err != nil {
		return err
	}

	db, err := store.NewDB("metrics" + types.String(this.itemConfig.Id))
	if err != nil {
		return err
	}

	{
		table, tableErr := kvstore.NewTable[*Stat]("items", &ItemEncoder[*Stat]{})
		if tableErr != nil {
			return tableErr
		}
		db.AddTable(table)
		this.itemsTable = table
	}

	{
		table, tableErr := kvstore.NewTable[[]byte]("values", kvstore.NewNilValueEncoder())
		if tableErr != nil {
			return tableErr
		}
		db.AddTable(table)
		this.valuesTable = table
	}

	{
		table, tableErr := kvstore.NewTable[[]byte]("sum_values", kvstore.NewBytesValueEncoder())
		if tableErr != nil {
			return tableErr
		}
		db.AddTable(table)
		this.sumTable = table
	}

	// 所有的服务IDs
	err = this.loadServerIdMap()
	if err != nil {
		return err
	}

	this.isLoaded = true

	return nil
}

func (this *KVTask) InsertStat(stat *Stat) error {
	if this.isStopped {
		return nil
	}
	if stat == nil {
		return nil
	}

	var version = this.itemConfig.Version

	this.serverIdMapLocker.Lock()
	this.serverTimeMap[types.String(stat.ServerId)+"_"+stat.Time] = zero.New()
	this.serverIdMapLocker.Unlock()

	if len(stat.Hash) == 0 {
		stat.Hash = stat.UniqueKey(version, this.itemConfig.Id)
	}

	var isNew bool
	var newValue = stat.Value

	// insert or update
	{
		var statKey = stat.FullKey(version, this.itemConfig.Id)
		oldStat, err := this.itemsTable.Get(statKey)
		var oldValue int64
		if err != nil {
			if !kvstore.IsNotFound(err) {
				return err
			}
			isNew = true
		} else {
			oldValue = oldStat.Value

			// delete old value from valuesTable
			err = this.valuesTable.Delete(oldStat.EncodeValueKey(version))
			if err != nil {
				return err
			}
		}

		oldValue += stat.Value
		stat.Value = oldValue
		err = this.itemsTable.Set(statKey, stat)
		if err != nil {
			return err
		}

		// insert value into valuesTable
		err = this.valuesTable.Insert(stat.EncodeValueKey(version), nil)
		if err != nil {
			return err
		}
	}

	// sum
	{
		var sumKey = stat.EncodeSumKey(version)
		sumResult, err := this.sumTable.Get(sumKey)
		var count uint64
		var total uint64
		if err != nil {
			if !kvstore.IsNotFound(err) {
				return err
			}
		} else {
			count, total = DecodeSumValue(sumResult)
		}

		if isNew {
			count++
		}
		total += uint64(newValue)

		err = this.sumTable.Set(sumKey, EncodeSumValue(count, total))
		if err != nil {
			return err
		}
	}

	return nil
}

func (this *KVTask) Upload(pauseDuration time.Duration) error {
	var uploadTr = trackers.Begin("METRIC:UPLOAD_STATS")
	defer uploadTr.End()

	if this.isStopped {
		return nil
	}

	this.serverIdMapLocker.Lock()

	// 服务IDs
	var serverTimeMap = this.serverTimeMap
	this.serverTimeMap = map[string]zero.Zero{} // 清空数据

	this.serverIdMapLocker.Unlock()

	if len(serverTimeMap) == 0 {
		return nil
	}

	// 控制缓存map不要太长
	if len(this.valuesCacheMap) > 4096 {
		var newMap = map[string]int64{}
		var countElements int
		for k, v := range this.valuesCacheMap {
			newMap[k] = v
			countElements++
			if countElements >= 2048 {
				break
			}
		}
		this.valuesCacheMap = newMap
	}

	// 开始上传
	rpcClient, err := rpc.SharedRPC()
	if err != nil {
		return err
	}

	var totalCount int

	for serverTime := range serverTimeMap {
		count, uploadErr := func(serverTime string) (int, error) {
			serverIdString, timeString, found := strings.Cut(serverTime, "_")
			if !found {
				return 0, nil
			}
			var serverId = types.Int64(serverIdString)
			if serverId <= 0 {
				return 0, nil
			}

			return this.uploadServerStats(rpcClient, serverId, timeString)
		}(serverTime)
		if uploadErr != nil {
			return uploadErr
		}

		totalCount += count

		// 休息一下，防止短时间内上传数据过多
		if pauseDuration > 0 && totalCount >= 100 {
			time.Sleep(pauseDuration)
			uploadTr.Add(-pauseDuration)
		}
	}

	return nil
}

func (this *KVTask) Start() error {
	// 读取数据
	this.statsTicker = utils.NewTicker(1 * time.Minute)
	if Tea.IsTesting() {
		this.statsTicker = utils.NewTicker(10 * time.Second)
	}
	goman.New(func() {
		for this.statsTicker.Next() {
			var tr = trackers.Begin("METRIC:DUMP_STATS_TO_LOCAL_DATABASE")

			this.statsLocker.Lock()
			var statsMap = this.statsMap
			this.statsMap = map[string]*Stat{}
			this.statsLocker.Unlock()

			for _, stat := range statsMap {
				err := this.InsertStat(stat)
				if err != nil {
					remotelogs.Error("METRIC", "insert stat failed: "+err.Error())
				}
			}

			tr.End()
		}
	})

	// 清理
	this.cleanTicker = time.NewTicker(24 * time.Hour)
	goman.New(func() {
		idles.RunTicker(this.cleanTicker, func() {
			var tr = trackers.Begin("METRIC:CLEAN_EXPIRED")
			err := this.CleanExpired()
			tr.End()
			if err != nil {
				remotelogs.Error("METRIC", "clean expired stats failed: "+err.Error())
			}
		})
	})

	// 上传
	this.uploadTicker = utils.NewTicker(this.itemConfig.UploadDuration())
	goman.New(func() {
		for this.uploadTicker.Next() {
			err := this.Upload(1 * time.Second)
			if err != nil && !rpc.IsConnError(err) {
				remotelogs.Error("METRIC", "upload stats failed: "+err.Error())
			}
		}
	})

	return nil
}

func (this *KVTask) Stop() error {
	this.isStopped = true

	if this.cleanTicker != nil {
		this.cleanTicker.Stop()
	}
	if this.uploadTicker != nil {
		this.uploadTicker.Stop()
	}
	if this.statsTicker != nil {
		this.statsTicker.Stop()
	}

	return nil
}

func (this *KVTask) Delete() error {
	this.isStopped = true

	return this.itemsTable.DB().Truncate()
}

func (this *KVTask) CleanExpired() error {
	if this.isStopped {
		return nil
	}

	var versionBytes = int32ToBigEndian(this.itemConfig.Version)
	var expiresTime = this.itemConfig.LocalExpiresTime()

	var rangeEnd = append([]byte(expiresTime+"_"), versionBytes...)
	rangeEnd = append(rangeEnd, 0xFF, 0xFF)

	{
		err := this.itemsTable.DeleteRange("", string(rangeEnd))
		if err != nil {
			return err
		}
	}

	{
		err := this.valuesTable.DeleteRange("", string(rangeEnd))
		if err != nil {
			return err
		}
	}

	{
		err := this.sumTable.DeleteRange("", string(rangeEnd))
		if err != nil {
			return err
		}
	}

	return nil
}

func (this *KVTask) Flush() error {
	return this.itemsTable.DB().Store().Flush()
}

func (this *KVTask) TestInspect(t *testing.T) {
	var db = this.itemsTable.DB()
	it, err := db.Store().RawDB().NewIter(&pebble.IterOptions{
		LowerBound: []byte(db.Namespace()),
		UpperBound: append([]byte(db.Namespace()), 0xFF),
	})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = it.Close()
	}()

	for it.First(); it.Valid(); it.Next() {
		valueBytes, valueErr := it.ValueAndErr()
		if valueErr != nil {
			t.Fatal(valueErr)
		}
		var key = string(it.Key()[len(db.Namespace())-1:])
		t.Log(key, "=>", string(valueBytes))
		if strings.HasPrefix(key, "$values$K$") {
			_, _, _, value, hash, _ := DecodeValueKey(key[len("$values$K$"):])
			t.Log("    |", hash, "=>", value)
		} else if strings.HasPrefix(key, "$sumValues$K$") {
			count, sum := DecodeSumValue(valueBytes)
			t.Log("    |", count, sum)
		}
	}
}

func (this *KVTask) Truncate() error {
	var db = this.itemsTable.DB()
	err := db.Truncate()
	if err != nil {
		return err
	}
	return db.Store().Flush()
}

func (this *KVTask) uploadServerStats(rpcClient *rpc.RPCClient, serverId int64, currentTime string) (countValues int, uploadErr error) {
	var pbStats []*pb.UploadingMetricStat
	var keepKeys []string

	var prefix = string(byteutils.Concat([]byte(currentTime), []byte{'_'}, int32ToBigEndian(this.itemConfig.Version), int64ToBigEndian(serverId)))
	var newCachedKeys = map[string]int64{}
	queryErr := this.valuesTable.
		Query().
		Prefix(prefix).
		Desc().
		Limit(20).
		FindAll(func(tx *kvstore.Tx[[]byte], item kvstore.Item[[]byte]) (goNext bool, err error) {
			_, _, version, value, hash, decodeErr := DecodeValueKey(item.Key)
			if decodeErr != nil {
				return false, decodeErr
			}
			if value <= 0 {
				return true, nil
			}

			// value not changed for the key
			if this.valuesCacheMap[hash] == value {
				keepKeys = append(keepKeys, hash)
				return true, nil
			}

			newCachedKeys[hash] = value

			stat, valueErr := this.itemsTable.Get(string(byteutils.Concat([]byte(currentTime), []byte{'_'}, int32ToBigEndian(version), []byte(hash))))
			if valueErr != nil {
				if kvstore.IsNotFound(valueErr) {
					return true, nil
				}
				return false, valueErr
			}
			if stat == nil {
				return true, nil
			}

			pbStats = append(pbStats, &pb.UploadingMetricStat{
				Id:    0, // not used in node
				Hash:  hash,
				Keys:  stat.Keys,
				Value: float32(value),
			})

			return true, nil
		})
	if queryErr != nil {
		return 0, queryErr
	}

	// count & total
	var count, total uint64
	{
		sumValue, err := this.sumTable.Get(prefix)
		if err != nil {
			if kvstore.IsNotFound(err) {
				return 0, nil
			}
			return 0, err
		}
		count, total = DecodeSumValue(sumValue)
	}

	_, err := rpcClient.MetricStatRPC.UploadMetricStats(rpcClient.Context(), &pb.UploadMetricStatsRequest{
		MetricStats: pbStats,
		Time:        currentTime,
		ServerId:    serverId,
		ItemId:      this.itemConfig.Id,
		Version:     this.itemConfig.Version,
		Count:       int64(count),
		Total:       float32(total),
		KeepKeys:    keepKeys,
	})
	if err != nil {
		return 0, err
	}

	// put into cache map MUST be after uploading success
	for k, v := range newCachedKeys {
		this.valuesCacheMap[k] = v
	}

	return len(pbStats), nil
}

func (this *KVTask) loadServerIdMap() error {
	var offsetKey string
	var currentTime = this.itemConfig.CurrentTime()
	for {
		var found bool
		err := this.sumTable.
			Query().
			Limit(1000).
			Offset(offsetKey).
			FindAll(func(tx *kvstore.Tx[[]byte], item kvstore.Item[[]byte]) (goNext bool, err error) {
				offsetKey = item.Key
				found = true

				serverId, timeString, version, decodeErr := DecodeSumKey(item.Key)
				if decodeErr != nil {
					return false, decodeErr
				}

				if version != this.itemConfig.Version || timeString != currentTime {
					return true, nil
				}

				this.serverIdMapLocker.Lock()
				this.serverTimeMap[types.String(serverId)+"_"+timeString] = zero.New()
				this.serverIdMapLocker.Unlock()

				return true, nil
			})
		if err != nil {
			return err
		}
		if !found {
			break
		}
	}

	return nil
}
