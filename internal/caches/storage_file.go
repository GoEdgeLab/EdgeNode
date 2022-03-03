package caches

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs"
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs/shared"
	"github.com/TeaOSLab/EdgeNode/internal/events"
	"github.com/TeaOSLab/EdgeNode/internal/goman"
	"github.com/TeaOSLab/EdgeNode/internal/remotelogs"
	"github.com/TeaOSLab/EdgeNode/internal/trackers"
	"github.com/TeaOSLab/EdgeNode/internal/utils"
	"github.com/TeaOSLab/EdgeNode/internal/zero"
	"github.com/iwind/TeaGo/Tea"
	"github.com/iwind/TeaGo/logs"
	"github.com/iwind/TeaGo/rands"
	"github.com/iwind/TeaGo/types"
	stringutil "github.com/iwind/TeaGo/utils/string"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

const (
	SizeExpiresAt    = 4
	SizeStatus       = 3
	SizeURLLength    = 4
	SizeHeaderLength = 4
	SizeBodyLength   = 8

	SizeMeta = SizeExpiresAt + SizeStatus + SizeURLLength + SizeHeaderLength + SizeBodyLength
)

const (
	HotItemSize = 1024
)

var sharedWritingFileKeyMap = map[string]zero.Zero{} // key => bool
var sharedWritingFileKeyLocker = sync.Mutex{}

// FileStorage 文件缓存
//   文件结构：
//    [expires time] | [ status ] | [url length] | [header length] | [body length] | [url] [header data] [body data]
type FileStorage struct {
	policy        *serverconfigs.HTTPCachePolicy
	cacheConfig   *serverconfigs.HTTPFileCacheStorage // 二级缓存
	memoryStorage *MemoryStorage                      // 一级缓存
	totalSize     int64

	list        ListInterface
	locker      sync.RWMutex
	purgeTicker *utils.Ticker

	hotMap       map[string]*HotItem // key => count
	hotMapLocker sync.Mutex
	lastHotSize  int
	hotTicker    *utils.Ticker

	openFileCache *OpenFileCache
}

func NewFileStorage(policy *serverconfigs.HTTPCachePolicy) *FileStorage {
	return &FileStorage{
		policy:      policy,
		hotMap:      map[string]*HotItem{},
		lastHotSize: -1,
	}
}

// Policy 获取当前的Policy
func (this *FileStorage) Policy() *serverconfigs.HTTPCachePolicy {
	return this.policy
}

// Init 初始化
func (this *FileStorage) Init() error {
	this.locker.Lock()
	defer this.locker.Unlock()

	before := time.Now()

	// 配置
	cacheConfig := &serverconfigs.HTTPFileCacheStorage{}
	optionsJSON, err := json.Marshal(this.policy.Options)
	if err != nil {
		return err
	}
	err = json.Unmarshal(optionsJSON, cacheConfig)
	if err != nil {
		return err
	}
	this.cacheConfig = cacheConfig

	if !filepath.IsAbs(this.cacheConfig.Dir) {
		this.cacheConfig.Dir = Tea.Root + Tea.DS + this.cacheConfig.Dir
	}

	this.cacheConfig.Dir = filepath.Clean(this.cacheConfig.Dir)
	var dir = this.cacheConfig.Dir

	if len(dir) == 0 {
		return errors.New("[CACHE]cache storage dir can not be empty")
	}

	list := NewFileList(dir + "/p" + strconv.FormatInt(this.policy.Id, 10))
	err = list.Init()
	if err != nil {
		return err
	}
	this.list = list
	stat, err := list.Stat(func(hash string) bool {
		return true
	})
	if err != nil {
		return err
	}
	this.totalSize = stat.Size
	this.list.OnAdd(func(item *Item) {
		atomic.AddInt64(&this.totalSize, item.TotalSize())
	})
	this.list.OnRemove(func(item *Item) {
		atomic.AddInt64(&this.totalSize, -item.TotalSize())
	})

	// 检查目录是否存在
	_, err = os.Stat(dir)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		} else {
			err = os.MkdirAll(dir, 0777)
			if err != nil {
				return errors.New("[CACHE]can not create dir:" + err.Error())
			}
		}
	}

	defer func() {
		// 统计
		count := stat.Count
		size := stat.Size

		cost := time.Since(before).Seconds() * 1000
		sizeMB := strconv.FormatInt(size, 10) + " Bytes"
		if size > 1024*1024*1024 {
			sizeMB = fmt.Sprintf("%.3f G", float64(size)/1024/1024/1024)
		} else if size > 1024*1024 {
			sizeMB = fmt.Sprintf("%.3f M", float64(size)/1024/1024)
		} else if size > 1024 {
			sizeMB = fmt.Sprintf("%.3f K", float64(size)/1024)
		}
		remotelogs.Println("CACHE", "init policy "+strconv.FormatInt(this.policy.Id, 10)+" from '"+this.cacheConfig.Dir+"', cost: "+fmt.Sprintf("%.2f", cost)+" ms, count: "+message.NewPrinter(language.English).Sprintf("%d", count)+", size: "+sizeMB)
	}()

	// 初始化list
	err = this.initList()
	if err != nil {
		return err
	}

	// 加载内存缓存
	if this.cacheConfig.MemoryPolicy != nil {
		if this.cacheConfig.MemoryPolicy.Capacity != nil && this.cacheConfig.MemoryPolicy.Capacity.Count > 0 {
			memoryPolicy := &serverconfigs.HTTPCachePolicy{
				Id:          this.policy.Id,
				IsOn:        this.policy.IsOn,
				Name:        this.policy.Name,
				Description: this.policy.Description,
				Capacity:    this.cacheConfig.MemoryPolicy.Capacity,
				MaxKeys:     this.policy.MaxKeys,
				MaxSize:     &shared.SizeCapacity{Count: 128, Unit: shared.SizeCapacityUnitMB}, // TODO 将来可以修改
				Type:        serverconfigs.CachePolicyStorageMemory,
				Options:     this.policy.Options,
				Life:        this.policy.Life,
				MinLife:     this.policy.MinLife,
				MaxLife:     this.policy.MaxLife,

				MemoryAutoPurgeCount:    this.policy.MemoryAutoPurgeCount,
				MemoryAutoPurgeInterval: this.policy.MemoryAutoPurgeInterval,
				MemoryLFUFreePercent:    this.policy.MemoryLFUFreePercent,
			}
			err = memoryPolicy.Init()
			if err != nil {
				return err
			}
			memoryStorage := NewMemoryStorage(memoryPolicy, this)
			err = memoryStorage.Init()
			if err != nil {
				return err
			}
			this.memoryStorage = memoryStorage
		}
	}

	// open file cache
	if this.cacheConfig.OpenFileCache != nil && this.cacheConfig.OpenFileCache.IsOn && this.cacheConfig.OpenFileCache.Max > 0 {
		this.openFileCache, err = NewOpenFileCache(this.cacheConfig.OpenFileCache.Max)
		logs.Println("start open file cache")
		if err != nil {
			remotelogs.Error("CACHE", "open file cache failed: "+err.Error())
		}
	}

	return nil
}

func (this *FileStorage) OpenReader(key string, useStale bool, isPartial bool) (Reader, error) {
	return this.openReader(key, true, useStale, isPartial)
}

func (this *FileStorage) openReader(key string, allowMemory bool, useStale bool, isPartial bool) (Reader, error) {
	// 使用陈旧缓存的时候，我们认为是短暂的，只需要从文件里检查即可
	if useStale {
		allowMemory = false
	}

	// 区间缓存只存在文件中
	if isPartial {
		allowMemory = false
	}

	// 先尝试内存缓存
	if allowMemory && this.memoryStorage != nil {
		reader, err := this.memoryStorage.OpenReader(key, useStale, isPartial)
		if err == nil {
			return reader, err
		}
	}

	hash, path := this.keyPath(key)

	// 检查文件记录是否已过期
	if !useStale {
		exists, err := this.list.Exist(hash)
		if err != nil {
			return nil, err
		}
		if !exists {
			return nil, ErrNotFound
		}
	}

	// TODO 尝试使用mmap加快读取速度
	var isOk = false
	var openFile *OpenFile
	if this.openFileCache != nil {
		openFile = this.openFileCache.Get(path)
	}
	var fp *os.File
	var err error
	if openFile == nil {
		fp, err = os.OpenFile(path, os.O_RDONLY, 0444)
		if err != nil {
			if !os.IsNotExist(err) {
				return nil, err
			}
			return nil, ErrNotFound
		}
	} else {
		fp = openFile.fp
	}
	defer func() {
		if !isOk {
			_ = fp.Close()
			_ = os.Remove(path)
		}
	}()

	var reader Reader
	if isPartial {
		var partialFileReader = NewPartialFileReader(fp)
		partialFileReader.openFile = openFile
		partialFileReader.openFileCache = this.openFileCache
		reader = partialFileReader
	} else {
		var fileReader = NewFileReader(fp)
		fileReader.openFile = openFile
		fileReader.openFileCache = this.openFileCache
		reader = fileReader
	}
	err = reader.Init()
	if err != nil {
		return nil, err
	}

	// 增加点击量
	// 1/1000采样
	if allowMemory {
		this.increaseHit(key, hash, reader)
	}

	isOk = true
	return reader, nil
}

// OpenWriter 打开缓存文件等待写入
func (this *FileStorage) OpenWriter(key string, expiredAt int64, status int, size int64, isPartial bool) (Writer, error) {
	// 先尝试内存缓存
	// 我们限定仅小文件优先存在内存中
	if !isPartial && this.memoryStorage != nil && size > 0 && size < 32*1024*1024 {
		writer, err := this.memoryStorage.OpenWriter(key, expiredAt, status, size, false)
		if err == nil {
			return writer, nil
		}
	}

	// 是否正在写入
	var isOk = false
	sharedWritingFileKeyLocker.Lock()
	_, ok := sharedWritingFileKeyMap[key]
	if ok {
		sharedWritingFileKeyLocker.Unlock()
		return nil, ErrFileIsWriting
	}
	sharedWritingFileKeyMap[key] = zero.New()
	sharedWritingFileKeyLocker.Unlock()
	defer func() {
		if !isOk {
			sharedWritingFileKeyLocker.Lock()
			delete(sharedWritingFileKeyMap, key)
			sharedWritingFileKeyLocker.Unlock()
		}
	}()

	// 检查是否超出最大值
	count, err := this.list.Count()
	if err != nil {
		return nil, err
	}
	if this.policy.MaxKeys > 0 && count > this.policy.MaxKeys {
		return nil, NewCapacityError("write file cache failed: too many keys in cache storage")
	}
	var capacityBytes = this.diskCapacityBytes()
	if capacityBytes > 0 && capacityBytes <= this.totalSize {
		return nil, NewCapacityError("write file cache failed: over disk size, current total size: " + strconv.FormatInt(this.totalSize, 10) + " bytes, capacity: " + strconv.FormatInt(capacityBytes, 10))
	}

	var hash = stringutil.Md5(key)
	var dir = this.cacheConfig.Dir + "/p" + strconv.FormatInt(this.policy.Id, 10) + "/" + hash[:2] + "/" + hash[2:4]
	_, err = os.Stat(dir)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
		err = os.MkdirAll(dir, 0777)
		if err != nil {
			return nil, err
		}
	}

	// 检查缓存是否已经生成
	var cachePathName = dir + "/" + hash
	var cachePath = cachePathName + ".cache"
	stat, err := os.Stat(cachePath)
	if err == nil && time.Now().Sub(stat.ModTime()) <= 1*time.Second {
		// 防止并发连续写入
		return nil, ErrFileIsWriting
	}
	var tmpPath = cachePath + ".tmp"
	if isPartial {
		tmpPath = cachePathName + ".cache"
	}

	// 先删除
	err = this.list.Remove(hash)
	if err != nil {
		return nil, err
	}

	writer, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_SYNC|os.O_WRONLY, 0666)
	if err != nil {
		return nil, err
	}

	var removeOnFailure = true
	defer func() {
		if err != nil {
			isOk = false
		}

		// 如果出错了，就删除文件，避免写一半
		if !isOk {
			_ = writer.Close()
			if removeOnFailure {
				_ = os.Remove(tmpPath)
			}
		}
	}()

	// 尝试锁定，如果锁定失败，则直接返回
	err = syscall.Flock(int(writer.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
	if err != nil {
		removeOnFailure = false
		return nil, ErrFileIsWriting
	}

	// 是否已经有内容
	var isNewCreated = true
	var partialBodyOffset int64
	if isPartial {
		partialFP, err := os.OpenFile(tmpPath, os.O_RDONLY, 0444)
		if err == nil {
			var partialReader = NewFileReader(partialFP)
			err = partialReader.InitAutoDiscard(false)
			if err == nil && partialReader.bodyOffset > 0 {
				isNewCreated = false
				partialBodyOffset = partialReader.bodyOffset
			}
			_ = partialReader.Close()
		}
	}

	if isNewCreated {
		err = writer.Truncate(0)
		if err != nil {
			return nil, err
		}

		// 写入过期时间
		bytes4 := make([]byte, 4)
		{
			binary.BigEndian.PutUint32(bytes4, uint32(expiredAt))
			_, err = writer.Write(bytes4)
			if err != nil {
				return nil, err
			}
		}

		// 写入状态码
		if status > 999 || status < 100 {
			status = 200
		}
		_, err = writer.WriteString(strconv.Itoa(status))
		if err != nil {
			return nil, err
		}

		// 写入URL长度
		{
			binary.BigEndian.PutUint32(bytes4, uint32(len(key)))
			_, err = writer.Write(bytes4)
			if err != nil {
				return nil, err
			}
		}

		// 写入Header Length
		{
			binary.BigEndian.PutUint32(bytes4, uint32(0))
			_, err = writer.Write(bytes4)
			if err != nil {
				return nil, err
			}
		}

		// 写入Body Length
		{
			b := make([]byte, SizeBodyLength)
			binary.BigEndian.PutUint64(b, uint64(0))
			_, err = writer.Write(b)
			if err != nil {
				return nil, err
			}
		}

		// 写入URL
		_, err = writer.WriteString(key)
		if err != nil {
			return nil, err
		}
	}

	isOk = true
	if isPartial {
		ranges, err := NewPartialRangesFromFile(cachePathName + "@ranges.cache")
		if err != nil {
			ranges = NewPartialRanges()
		}

		return NewPartialFileWriter(writer, key, expiredAt, isNewCreated, isPartial, partialBodyOffset, ranges, func() {
			sharedWritingFileKeyLocker.Lock()
			delete(sharedWritingFileKeyMap, key)
			sharedWritingFileKeyLocker.Unlock()
		}), nil
	} else {
		return NewFileWriter(writer, key, expiredAt, func() {
			sharedWritingFileKeyLocker.Lock()
			delete(sharedWritingFileKeyMap, key)
			sharedWritingFileKeyLocker.Unlock()
		}), nil
	}
}

// AddToList 添加到List
func (this *FileStorage) AddToList(item *Item) {
	if this.memoryStorage != nil {
		if item.Type == ItemTypeMemory {
			this.memoryStorage.AddToList(item)
			return
		}
	}

	item.MetaSize = SizeMeta + 128
	hash := stringutil.Md5(item.Key)
	err := this.list.Add(hash, item)
	if err != nil && !strings.Contains(err.Error(), "UNIQUE constraint failed") {
		remotelogs.Error("CACHE", "add to list failed: "+err.Error())
	}
}

// Delete 删除某个键值对应的缓存
func (this *FileStorage) Delete(key string) error {
	this.locker.Lock()
	defer this.locker.Unlock()

	// 先尝试内存缓存
	if this.memoryStorage != nil {
		_ = this.memoryStorage.Delete(key)
	}

	hash, path := this.keyPath(key)
	err := this.list.Remove(hash)
	if err != nil {
		return err
	}
	err = os.Remove(path)
	if err == nil || os.IsNotExist(err) {
		return nil
	}
	return err
}

// Stat 统计
func (this *FileStorage) Stat() (*Stat, error) {
	this.locker.RLock()
	defer this.locker.RUnlock()

	return this.list.Stat(func(hash string) bool {
		return true
	})
}

// CleanAll 清除所有的缓存
func (this *FileStorage) CleanAll() error {
	this.locker.Lock()
	defer this.locker.Unlock()

	// 先尝试内存缓存
	if this.memoryStorage != nil {
		_ = this.memoryStorage.CleanAll()
	}

	err := this.list.CleanAll()
	if err != nil {
		return err
	}

	// 删除缓存和目录
	// 不能直接删除子目录，比较危险
	dir := this.dir()
	fp, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer func() {
		_ = fp.Close()
	}()

	stat, err := fp.Stat()
	if err != nil {
		return err
	}

	if !stat.IsDir() {
		return nil
	}

	// 改成待删除
	subDirs, err := fp.Readdir(-1)
	if err != nil {
		return err
	}
	for _, info := range subDirs {
		subDir := info.Name()

		// 检查目录名
		ok, err := regexp.MatchString(`^[0-9a-f]{2}$`, subDir)
		if err != nil {
			return err
		}
		if !ok {
			continue
		}

		// 修改目录名
		tmpDir := dir + "/" + subDir + "-deleted"
		err = os.Rename(dir+"/"+subDir, tmpDir)
		if err != nil {
			return err
		}
	}

	// 重新遍历待删除
	goman.New(func() {
		err = this.cleanDeletedDirs(dir)
		if err != nil {
			remotelogs.Warn("CACHE", "delete '*-deleted' dirs failed: "+err.Error())
		}
	})

	return nil
}

// Purge 清理过期的缓存
func (this *FileStorage) Purge(keys []string, urlType string) error {
	this.locker.Lock()
	defer this.locker.Unlock()

	// 先尝试内存缓存
	if this.memoryStorage != nil {
		_ = this.memoryStorage.Purge(keys, urlType)
	}

	// 目录
	if urlType == "dir" {
		for _, key := range keys {
			err := this.list.CleanPrefix(key)
			if err != nil {
				return err
			}
		}
	}

	// 文件
	for _, key := range keys {
		hash, path := this.keyPath(key)
		err := os.Remove(path)
		if err != nil && !os.IsNotExist(err) {
			return err
		}
		err = this.list.Remove(hash)
		if err != nil {
			return err
		}
	}
	return nil
}

// Stop 停止
func (this *FileStorage) Stop() {
	events.Remove(this)

	this.locker.Lock()
	defer this.locker.Unlock()

	// 先尝试内存缓存
	if this.memoryStorage != nil {
		this.memoryStorage.Stop()
	}

	_ = this.list.Reset()
	if this.purgeTicker != nil {
		this.purgeTicker.Stop()
	}
	if this.hotTicker != nil {
		this.hotTicker.Stop()
	}

	_ = this.list.Close()

	if this.openFileCache != nil {
		this.openFileCache.CloseAll()
	}
}

// TotalDiskSize 消耗的磁盘尺寸
func (this *FileStorage) TotalDiskSize() int64 {
	return atomic.LoadInt64(&this.totalSize)
}

// TotalMemorySize 内存尺寸
func (this *FileStorage) TotalMemorySize() int64 {
	if this.memoryStorage == nil {
		return 0
	}
	return this.memoryStorage.TotalMemorySize()
}

// 绝对路径
func (this *FileStorage) dir() string {
	return this.cacheConfig.Dir + "/p" + strconv.FormatInt(this.policy.Id, 10) + "/"
}

// 获取Key对应的文件路径
func (this *FileStorage) keyPath(key string) (hash string, path string) {
	hash = stringutil.Md5(key)
	dir := this.cacheConfig.Dir + "/p" + strconv.FormatInt(this.policy.Id, 10) + "/" + hash[:2] + "/" + hash[2:4]
	path = dir + "/" + hash + ".cache"
	return
}

// 获取Hash对应的文件路径
func (this *FileStorage) hashPath(hash string) (path string) {
	if len(hash) != 32 {
		return ""
	}
	dir := this.cacheConfig.Dir + "/p" + strconv.FormatInt(this.policy.Id, 10) + "/" + hash[:2] + "/" + hash[2:4]
	path = dir + "/" + hash + ".cache"
	return
}

// 初始化List
func (this *FileStorage) initList() error {
	err := this.list.Reset()
	if err != nil {
		return err
	}

	// 使用异步防止阻塞主线程
	/**goman.New(func() {
		dir := this.dir()

		// 清除tmp
		// TODO 需要一个更加高效的实现
	})**/

	// 启动定时清理任务
	var autoPurgeInterval = this.policy.PersistenceAutoPurgeInterval
	if autoPurgeInterval <= 0 {
		autoPurgeInterval = 30
		if Tea.IsTesting() {
			autoPurgeInterval = 10
		}
	}
	this.purgeTicker = utils.NewTicker(time.Duration(autoPurgeInterval) * time.Second)
	events.OnKey(events.EventQuit, this, func() {
		remotelogs.Println("CACHE", "quit clean timer")

		{
			var ticker = this.purgeTicker
			if ticker != nil {
				ticker.Stop()
			}
		}
		{
			var ticker = this.hotTicker
			if ticker != nil {
				ticker.Stop()
			}
		}
	})
	goman.New(func() {
		for this.purgeTicker.Next() {
			trackers.Run("FILE_CACHE_STORAGE_PURGE_LOOP", func() {
				this.purgeLoop()
			})
		}
	})

	// 热点处理任务
	this.hotTicker = utils.NewTicker(1 * time.Minute)
	if Tea.IsTesting() {
		this.hotTicker = utils.NewTicker(10 * time.Second)
	}
	goman.New(func() {
		for this.hotTicker.Next() {
			trackers.Run("FILE_CACHE_STORAGE_HOT_LOOP", func() {
				this.hotLoop()
			})
		}
	})

	return nil
}

// 清理任务
func (this *FileStorage) purgeLoop() {
	// 计算是否应该开启LFU清理
	var capacityBytes = this.policy.CapacityBytes()
	var startLFU = false
	var usedPercent = float32(this.TotalDiskSize()*100) / float32(capacityBytes)
	var lfuFreePercent = this.policy.PersistenceLFUFreePercent
	if lfuFreePercent <= 0 {
		lfuFreePercent = 5
	}
	if capacityBytes > 0 {
		if lfuFreePercent < 100 {
			if usedPercent >= 100-lfuFreePercent {
				startLFU = true
			}
		}
	}

	// 清理过期
	{
		var times = 1

		// 空闲时间多清理
		if utils.SharedFreeHoursManager.IsFreeHour() {
			times = 5
		}

		// 处于LFU阈值时，多清理
		if startLFU {
			times = 5
		}

		var purgeCount = this.policy.PersistenceAutoPurgeCount
		if purgeCount <= 0 {
			purgeCount = 1000
		}
		for i := 0; i < times; i++ {
			countFound, err := this.list.Purge(purgeCount, func(hash string) error {
				path := this.hashPath(hash)
				err := os.Remove(path)
				if err != nil && !os.IsNotExist(err) {
					remotelogs.Error("CACHE", "purge '"+path+"' error: "+err.Error())
				}
				return nil
			})
			if err != nil {
				remotelogs.Warn("CACHE", "purge file storage failed: "+err.Error())
				continue
			}

			if countFound < purgeCount {
				break
			}

			time.Sleep(1 * time.Second)
		}
	}

	// 磁盘空间不足时，清除老旧的缓存
	if startLFU {
		var total, _ = this.list.Count()
		if total > 0 {
			var count = types.Int(math.Ceil(float64(total) * float64(lfuFreePercent*2) / 100))
			if count > 0 {
				// 限制单次清理的条数，防止占用太多系统资源
				if count > 2000 {
					count = 2000
				}

				remotelogs.Println("CACHE", "LFU purge policy '"+this.policy.Name+"' id: "+types.String(this.policy.Id)+", count: "+types.String(count))
				err := this.list.PurgeLFU(count, func(hash string) error {
					path := this.hashPath(hash)
					err := os.Remove(path)
					if err != nil && !os.IsNotExist(err) {
						remotelogs.Error("CACHE", "purge '"+path+"' error: "+err.Error())
					}
					return nil
				})
				if err != nil {
					remotelogs.Warn("CACHE", "purge file storage in LFU failed: "+err.Error())
				}
			}
		}
	}
}

// 热点数据任务
func (this *FileStorage) hotLoop() {
	var memoryStorage = this.memoryStorage
	if memoryStorage == nil {
		return
	}

	this.hotMapLocker.Lock()
	if len(this.hotMap) == 0 {
		this.hotMapLocker.Unlock()
		this.lastHotSize = 0
		return
	}

	this.lastHotSize = len(this.hotMap)

	var result = []*HotItem{} // [ {key: ..., hits: ...}, ... ]
	for _, v := range this.hotMap {
		result = append(result, v)
	}

	this.hotMap = map[string]*HotItem{}
	this.hotMapLocker.Unlock()

	// 取Top10写入内存
	if len(result) > 0 {
		sort.Slice(result, func(i, j int) bool {
			return result[i].Hits > result[j].Hits
		})
		var size = 1
		if len(result) < 10 {
			size = 1
		} else {
			size = len(result) / 10
		}

		var buf = utils.BytePool16k.Get()
		defer utils.BytePool16k.Put(buf)
		for _, item := range result[:size] {
			reader, err := this.openReader(item.Key, false, false, false)
			if err != nil {
				continue
			}
			if reader == nil {
				continue
			}
			if reader.ExpiresAt() <= time.Now().Unix() {
				continue
			}

			writer, err := this.memoryStorage.openWriter(item.Key, item.ExpiresAt, item.Status, reader.BodySize(), false)
			if err != nil {
				if !CanIgnoreErr(err) {
					remotelogs.Error("CACHE", "transfer hot item failed: "+err.Error())
				}
				_ = reader.Close()
				continue
			}
			if writer == nil {
				_ = reader.Close()
				continue
			}

			err = reader.ReadHeader(buf, func(n int) (goNext bool, err error) {
				_, err = writer.WriteHeader(buf[:n])
				return
			})
			if err != nil {
				_ = reader.Close()
				_ = writer.Discard()
				continue
			}

			err = reader.ReadBody(buf, func(n int) (goNext bool, err error) {
				_, err = writer.Write(buf[:n])
				if err == nil {
					goNext = true
				}
				return
			})
			if err != nil {
				_ = reader.Close()
				_ = writer.Discard()
				continue
			}

			this.memoryStorage.AddToList(&Item{
				Type:       writer.ItemType(),
				Key:        item.Key,
				ExpiredAt:  item.ExpiresAt,
				HeaderSize: writer.HeaderSize(),
				BodySize:   writer.BodySize(),
			})

			_ = reader.Close()
			_ = writer.Close()
		}
	}
}

func (this *FileStorage) diskCapacityBytes() int64 {
	c1 := this.policy.CapacityBytes()
	if SharedManager.MaxDiskCapacity != nil {
		c2 := SharedManager.MaxDiskCapacity.Bytes()
		if c2 > 0 {
			return c2
		}
	}
	return c1
}

// 清理 *-deleted 目录
// 由于在很多硬盘上耗时非常久，所以应该放在后台运行
func (this *FileStorage) cleanDeletedDirs(dir string) error {
	fp, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer func() {
		_ = fp.Close()
	}()
	subDirs, err := fp.Readdir(-1)
	if err != nil {
		return err
	}
	for _, info := range subDirs {
		subDir := info.Name()
		if !strings.HasSuffix(subDir, "-deleted") {
			continue
		}

		// 删除
		err = os.RemoveAll(dir + "/" + subDir)
		if err != nil {
			if !os.IsNotExist(err) {
				return err
			}
		}
	}
	return nil
}

// 增加某个Key的点击量
func (this *FileStorage) increaseHit(key string, hash string, reader Reader) {
	var rate = this.policy.PersistenceHitSampleRate
	if rate <= 0 {
		rate = 1000
	}
	if this.lastHotSize == 0 {
		// 自动降低采样率来增加热点数据的缓存几率
		rate = rate / 10
	}
	if rands.Int(0, rate) == 0 {
		var hitErr = this.list.IncreaseHit(hash)
		if hitErr != nil {
			// 此错误可以忽略
			remotelogs.Error("CACHE", "increase hit failed: "+hitErr.Error())
		}

		// 增加到热点
		// 这里不收录缓存尺寸过大的文件
		if this.memoryStorage != nil && reader.BodySize() > 0 && reader.BodySize() < 128*1024*1024 {
			this.hotMapLocker.Lock()
			hotItem, ok := this.hotMap[key]
			if ok {
				hotItem.Hits++
				hotItem.ExpiresAt = reader.ExpiresAt()
			} else if len(this.hotMap) < HotItemSize { // 控制数量
				this.hotMap[key] = &HotItem{
					Key:       key,
					ExpiresAt: reader.ExpiresAt(),
					Status:    reader.Status(),
					Hits:      1,
				}
			}
			this.hotMapLocker.Unlock()
		}
	}
}
