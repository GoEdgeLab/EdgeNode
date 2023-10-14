package caches

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs"
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs/shared"
	teaconst "github.com/TeaOSLab/EdgeNode/internal/const"
	"github.com/TeaOSLab/EdgeNode/internal/events"
	"github.com/TeaOSLab/EdgeNode/internal/goman"
	"github.com/TeaOSLab/EdgeNode/internal/remotelogs"
	"github.com/TeaOSLab/EdgeNode/internal/trackers"
	"github.com/TeaOSLab/EdgeNode/internal/utils"
	"github.com/TeaOSLab/EdgeNode/internal/utils/fasttime"
	fsutils "github.com/TeaOSLab/EdgeNode/internal/utils/fs"
	setutils "github.com/TeaOSLab/EdgeNode/internal/utils/sets"
	"github.com/TeaOSLab/EdgeNode/internal/utils/sizes"
	"github.com/TeaOSLab/EdgeNode/internal/zero"
	"github.com/iwind/TeaGo/Tea"
	"github.com/iwind/TeaGo/rands"
	"github.com/iwind/TeaGo/types"
	stringutil "github.com/iwind/TeaGo/utils/string"
	timeutil "github.com/iwind/TeaGo/utils/time"
	"github.com/iwind/gosock/pkg/gosock"
	"github.com/shirou/gopsutil/v3/load"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

const (
	SizeExpiresAt      = 4
	OffsetExpiresAt    = 0
	SizeStatus         = 3
	OffsetStatus       = SizeExpiresAt
	SizeURLLength      = 4
	OffsetURLLength    = OffsetStatus + SizeStatus
	SizeHeaderLength   = 4
	OffsetHeaderLength = OffsetURLLength + SizeURLLength
	SizeBodyLength     = 8
	OffsetBodyLength   = OffsetHeaderLength + SizeHeaderLength

	SizeMeta = SizeExpiresAt + SizeStatus + SizeURLLength + SizeHeaderLength + SizeBodyLength
)

const (
	FileStorageMaxIgnoreKeys        = 32768        // 最大可忽略的键值数（尺寸过大的键值）
	HotItemSize                     = 1024         // 热点数据数量
	HotItemLifeSeconds       int64  = 3600         // 热点数据生命周期
	FileToMemoryMaxSize             = 32 * sizes.M // 可以从文件写入到内存的最大文件尺寸
	FileTmpSuffix                   = ".tmp"
	DefaultMinDiskFreeSpace  uint64 = 5 << 30 // 当前磁盘最小剩余空间
	DefaultStaleCacheSeconds        = 1200    // 过时缓存留存时间
	HashKeyLength                   = 32
)

var sharedWritingFileKeyMap = map[string]zero.Zero{} // key => bool
var sharedWritingFileKeyLocker = sync.Mutex{}

// FileStorage 文件缓存
//
//	文件结构：
//	 [expires time] | [ status ] | [url length] | [header length] | [body length] | [url] [header data] [body data]
type FileStorage struct {
	policy        *serverconfigs.HTTPCachePolicy
	options       *serverconfigs.HTTPFileCacheStorage // 二级缓存
	memoryStorage *MemoryStorage                      // 一级缓存

	list        ListInterface
	locker      sync.RWMutex
	purgeTicker *utils.Ticker

	hotMap       map[string]*HotItem // key => count
	hotMapLocker sync.Mutex
	lastHotSize  int
	hotTicker    *utils.Ticker

	ignoreKeys *setutils.FixedSet

	openFileCache *OpenFileCache

	mainDiskIsFull    bool
	mainDiskTotalSize uint64

	subDirs []*FileDir
}

func NewFileStorage(policy *serverconfigs.HTTPCachePolicy) *FileStorage {
	return &FileStorage{
		policy:      policy,
		hotMap:      map[string]*HotItem{},
		lastHotSize: -1,
		ignoreKeys:  setutils.NewFixedSet(FileStorageMaxIgnoreKeys),
	}
}

// Policy 获取当前的Policy
func (this *FileStorage) Policy() *serverconfigs.HTTPCachePolicy {
	return this.policy
}

// CanUpdatePolicy 检查策略是否可以更新
func (this *FileStorage) CanUpdatePolicy(newPolicy *serverconfigs.HTTPCachePolicy) bool {
	// 检查路径是否有变化
	oldOptionsJSON, err := json.Marshal(this.policy.Options)
	if err != nil {
		return false
	}
	var oldOptions = &serverconfigs.HTTPFileCacheStorage{}
	err = json.Unmarshal(oldOptionsJSON, oldOptions)
	if err != nil {
		return false
	}

	newOptionsJSON, err := json.Marshal(newPolicy.Options)
	if err != nil {
		return false
	}
	var newOptions = &serverconfigs.HTTPFileCacheStorage{}
	err = json.Unmarshal(newOptionsJSON, newOptions)
	if err != nil {
		return false
	}

	if oldOptions.Dir == newOptions.Dir {
		return true
	}

	return false
}

// UpdatePolicy 修改策略
func (this *FileStorage) UpdatePolicy(newPolicy *serverconfigs.HTTPCachePolicy) {
	var oldOpenFileCache = this.options.OpenFileCache

	this.policy = newPolicy

	newOptionsJSON, err := json.Marshal(newPolicy.Options)
	if err != nil {
		return
	}
	var newOptions = &serverconfigs.HTTPFileCacheStorage{}
	err = json.Unmarshal(newOptionsJSON, newOptions)
	if err != nil {
		remotelogs.Error("CACHE", "update policy '"+types.String(this.policy.Id)+"' failed: decode options failed: "+err.Error())
		return
	}

	var subDirs = []*FileDir{}
	for _, subDir := range newOptions.SubDirs {
		subDirs = append(subDirs, &FileDir{
			Path:     subDir.Path,
			Capacity: subDir.Capacity,
			IsFull:   false,
		})
	}
	this.subDirs = subDirs
	this.checkDiskSpace()

	err = newOptions.Init()
	if err != nil {
		remotelogs.Error("CACHE", "update policy '"+types.String(this.policy.Id)+"' failed: init options failed: "+err.Error())
		return
	}

	this.options = newOptions

	var memoryStorage = this.memoryStorage
	if memoryStorage != nil {
		if newOptions.MemoryPolicy != nil && newOptions.MemoryPolicy.CapacityBytes() > 0 {
			memoryStorage.UpdatePolicy(newOptions.MemoryPolicy)
		} else {
			memoryStorage.Stop()
			this.memoryStorage = nil
		}
	} else if newOptions.MemoryPolicy != nil && this.options.MemoryPolicy.Capacity != nil && this.options.MemoryPolicy.Capacity.Count > 0 {
		err = this.createMemoryStorage()
		if err != nil {
			remotelogs.Error("CACHE", "update policy '"+types.String(this.policy.Id)+"' failed: create memory storage failed: "+err.Error())
		}
	}

	// open cache
	oldOpenFileCacheJSON, _ := json.Marshal(oldOpenFileCache)
	newOpenFileCacheJSON, _ := json.Marshal(this.options.OpenFileCache)
	if !bytes.Equal(oldOpenFileCacheJSON, newOpenFileCacheJSON) {
		this.initOpenFileCache()
	}

	// Purge Ticker
	if newPolicy.PersistenceAutoPurgeInterval != this.policy.PersistenceAutoPurgeInterval {
		this.initPurgeTicker()
	}

	// reset ignored keys
	this.ignoreKeys.Reset()
}

// Init 初始化
func (this *FileStorage) Init() error {
	this.locker.Lock()
	defer this.locker.Unlock()

	var before = time.Now()

	// 配置
	var options = &serverconfigs.HTTPFileCacheStorage{}
	optionsJSON, err := json.Marshal(this.policy.Options)
	if err != nil {
		return err
	}
	err = json.Unmarshal(optionsJSON, options)
	if err != nil {
		return err
	}
	this.options = options

	if !filepath.IsAbs(this.options.Dir) {
		this.options.Dir = Tea.Root + Tea.DS + this.options.Dir
	}

	this.options.Dir = filepath.Clean(this.options.Dir)
	var dir = this.options.Dir

	var subDirs = []*FileDir{}
	for _, subDir := range this.options.SubDirs {
		subDirs = append(subDirs, &FileDir{
			Path:     subDir.Path,
			Capacity: subDir.Capacity,
			IsFull:   false,
		})
	}
	this.subDirs = subDirs
	if len(subDirs) > 0 {
		this.checkDiskSpace()
	}

	if len(dir) == 0 {
		return errors.New("[CACHE]cache storage dir can not be empty")
	}

	var list = NewFileList(dir + "/p" + types.String(this.policy.Id) + "/.indexes")
	err = list.Init()
	if err != nil {
		return err
	}
	list.(*FileList).SetOldDir(dir + "/p" + types.String(this.policy.Id))
	this.list = list

	// 检查目录是否存在
	_, err = os.Stat(dir)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		} else {
			err = os.MkdirAll(dir, 0777)
			if err != nil {
				return fmt.Errorf("[CACHE]can not create dir: %w", err)
			}
		}
	}

	defer func() {
		// 统计
		var totalSize = this.TotalDiskSize()
		var cost = time.Since(before).Seconds() * 1000
		var sizeMB = types.String(totalSize) + " Bytes"
		if totalSize > 1*sizes.G {
			sizeMB = fmt.Sprintf("%.3f G", float64(totalSize)/float64(sizes.G))
		} else if totalSize > 1*sizes.M {
			sizeMB = fmt.Sprintf("%.3f M", float64(totalSize)/float64(sizes.M))
		} else if totalSize > 1*sizes.K {
			sizeMB = fmt.Sprintf("%.3f K", float64(totalSize)/float64(sizes.K))
		}
		remotelogs.Println("CACHE", "init policy "+types.String(this.policy.Id)+" from '"+this.options.Dir+"', cost: "+fmt.Sprintf("%.2f", cost)+" ms, size: "+sizeMB)
	}()

	// 初始化list
	err = this.initList()
	if err != nil {
		return err
	}

	// 加载内存缓存
	if this.options.MemoryPolicy != nil && this.options.MemoryPolicy.Capacity != nil && this.options.MemoryPolicy.Capacity.Count > 0 {
		err = this.createMemoryStorage()
		if err != nil {
			return err
		}
	}

	// open file cache
	this.initOpenFileCache()

	// 检查磁盘空间
	this.checkDiskSpace()

	// clean *.trash directories
	this.cleanAllDeletedDirs()

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
	var memoryStorage = this.memoryStorage
	if allowMemory && memoryStorage != nil {
		reader, err := memoryStorage.OpenReader(key, useStale, isPartial)
		if err == nil {
			return reader, err
		}
	}

	hash, path, _ := this.keyPath(key)

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
	var openFileCache = this.openFileCache // 因为中间可能有修改，所以先赋值再获取
	if openFileCache != nil {
		openFile = openFileCache.Get(path)
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
			_ = this.removeCacheFile(path)
		}
	}()

	var reader Reader
	if isPartial {
		var partialFileReader = NewPartialFileReader(fp)
		partialFileReader.openFile = openFile
		partialFileReader.openFileCache = openFileCache
		reader = partialFileReader
	} else {
		var fileReader = NewFileReader(fp)
		fileReader.openFile = openFile
		fileReader.openFileCache = openFileCache
		reader = fileReader
	}
	err = reader.Init()
	if err != nil {
		return nil, err
	}

	// 增加点击量
	// 1/1000采样
	if !isPartial && allowMemory && reader.BodySize() < FileToMemoryMaxSize {
		this.increaseHit(key, hash, reader)
	}

	isOk = true
	return reader, nil
}

// OpenWriter 打开缓存文件等待写入
func (this *FileStorage) OpenWriter(key string, expiresAt int64, status int, headerSize int, bodySize int64, maxSize int64, isPartial bool) (Writer, error) {
	return this.openWriter(key, expiresAt, status, headerSize, bodySize, maxSize, isPartial, false)
}

// OpenFlushWriter 打开从其他媒介直接刷入的写入器
func (this *FileStorage) OpenFlushWriter(key string, expiresAt int64, status int, headerSize int, bodySize int64) (Writer, error) {
	return this.openWriter(key, expiresAt, status, headerSize, bodySize, -1, false, true)
}

func (this *FileStorage) openWriter(key string, expiredAt int64, status int, headerSize int, bodySize int64, maxSize int64, isPartial bool, isFlushing bool) (Writer, error) {
	// 是否正在退出
	if teaconst.IsQuiting {
		return nil, ErrWritingUnavailable
	}

	// 是否已忽略
	if maxSize > 0 && this.ignoreKeys.Has(types.String(maxSize)+"$"+key) {
		return nil, ErrEntityTooLarge
	}

	// 检查磁盘是否超出容量
	// 需要在内存缓存之前执行，避免成功写进到了内存缓存，但无法刷到磁盘
	var capacityBytes = this.diskCapacityBytes()
	if capacityBytes > 0 && capacityBytes <= this.TotalDiskSize()+(32<<20 /** 余量 **/) {
		return nil, NewCapacityError("write file cache failed: over disk size, current: " + types.String(this.TotalDiskSize()) + ", capacity: " + types.String(capacityBytes))
	}

	// 先尝试内存缓存
	// 我们限定仅小文件优先存在内存中
	var maxMemorySize = FileToMemoryMaxSize
	if maxSize > 0 && maxSize < maxMemorySize {
		maxMemorySize = maxSize
	}
	var memoryStorage = this.memoryStorage
	if !fsutils.DiskIsExtremelyFast() && !isFlushing && !isPartial && memoryStorage != nil && ((bodySize > 0 && bodySize < maxMemorySize) || bodySize < 0) {
		writer, err := memoryStorage.OpenWriter(key, expiredAt, status, headerSize, bodySize, maxMemorySize, false)
		if err == nil {
			return writer, nil
		}

		// 如果队列满了，则等待
		if errors.Is(err, ErrWritingQueueFull) {
			return nil, err
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

	if !isFlushing && !fsutils.WriteReady() {
		sharedWritingFileKeyLocker.Unlock()
		return nil, ErrServerIsBusy
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

	var hash = stringutil.Md5(key)

	dir, diskIsFull := this.subDir(hash)
	if diskIsFull {
		return nil, NewCapacityError("the disk is full")
	}

	// 检查缓存是否已经生成
	var cachePathName = dir + "/" + hash
	var cachePath = cachePathName + ".cache"

	// 关闭OpenFileCache
	var openFileCache = this.openFileCache
	if openFileCache != nil {
		openFileCache.Close(cachePath)
	}

	// 查询当前已有缓存文件
	stat, err := os.Stat(cachePath)

	// 检查两次写入缓存的时间是否过于相近，分片内容不受此限制
	if err == nil && !isPartial && time.Since(stat.ModTime()) <= 1*time.Second {
		// 防止并发连续写入
		return nil, ErrFileIsWriting
	}

	// 构造文件名
	var tmpPath = cachePath
	var existsFile = false
	if stat != nil {
		existsFile = true

		// 如果已经存在，则增加一个.tmp后缀，防止读写冲突
		tmpPath += FileTmpSuffix
	}

	if isPartial {
		tmpPath = cachePathName + ".cache"
	}

	// 先删除
	if !isPartial {
		err = this.list.Remove(hash)
		if err != nil {
			return nil, err
		}
	}

	// 从已经存储的内容中读取信息
	var isNewCreated = true
	var partialBodyOffset int64
	var partialRanges *PartialRanges
	if isPartial {
		// 数据库中是否存在
		existsCacheItem, _ := this.list.Exist(hash)
		if existsCacheItem {
			readerFp, err := os.OpenFile(tmpPath, os.O_RDONLY, 0444)
			if err == nil {
				var partialReader = NewPartialFileReader(readerFp)
				err = partialReader.Init()
				_ = partialReader.Close()
				if err == nil && partialReader.bodyOffset > 0 {
					partialRanges = partialReader.Ranges()
					if bodySize > 0 && partialRanges != nil && partialRanges.BodySize > 0 && bodySize != partialRanges.BodySize {
						_ = this.removeCacheFile(tmpPath)
					} else {
						isNewCreated = false
						partialBodyOffset = partialReader.bodyOffset
					}
				} else {
					_ = this.removeCacheFile(tmpPath)
				}
			}
		}
		if isNewCreated {
			err = this.list.Remove(hash)
			if err != nil {
				return nil, err
			}
		}
		if partialRanges == nil {
			partialRanges = NewPartialRanges(expiredAt)
		}
	}

	var flags = os.O_CREATE | os.O_WRONLY
	if isNewCreated && existsFile {
		flags |= os.O_TRUNC
	}
	fsutils.WriteBegin()
	writer, err := os.OpenFile(tmpPath, flags, 0666)
	fsutils.WriteEnd()
	if err != nil {
		// TODO 检查在各个系统中的稳定性
		if os.IsNotExist(err) {
			_ = os.MkdirAll(dir, 0777)

			// open file again
			writer, err = os.OpenFile(tmpPath, flags, 0666)
		}
		if err != nil {
			return nil, err
		}
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

	var metaBodySize int64 = -1
	var metaHeaderSize = -1
	if isNewCreated {
		// 写入meta
		// 从v0.5.8开始不再在meta中写入Key
		var metaBytes = make([]byte, SizeMeta)
		binary.BigEndian.PutUint32(metaBytes[OffsetExpiresAt:], uint32(expiredAt))

		// 写入状态码
		if status > 999 || status < 100 {
			status = 200
		}
		copy(metaBytes[OffsetStatus:], strconv.Itoa(status))

		// 写入Header Length
		if headerSize > 0 {
			binary.BigEndian.PutUint32(metaBytes[OffsetHeaderLength:], uint32(headerSize))
			metaHeaderSize = headerSize
		}

		// 写入Body Length
		if bodySize > 0 {
			binary.BigEndian.PutUint64(metaBytes[OffsetBodyLength:], uint64(bodySize))
			metaBodySize = bodySize
		}

		fsutils.WriteBegin()
		_, err = writer.Write(metaBytes)
		fsutils.WriteEnd()
		if err != nil {
			return nil, err
		}
	}

	isOk = true
	if isPartial {
		return NewPartialFileWriter(writer, key, expiredAt, metaHeaderSize, metaBodySize, isNewCreated, isPartial, partialBodyOffset, partialRanges, func() {
			sharedWritingFileKeyLocker.Lock()
			delete(sharedWritingFileKeyMap, key)
			sharedWritingFileKeyLocker.Unlock()
		}), nil
	} else {
		return NewFileWriter(this, writer, key, expiredAt, metaHeaderSize, metaBodySize, maxSize, func() {
			sharedWritingFileKeyLocker.Lock()
			delete(sharedWritingFileKeyMap, key)
			sharedWritingFileKeyLocker.Unlock()
		}), nil
	}
}

// AddToList 添加到List
func (this *FileStorage) AddToList(item *Item) {
	// 是否正在退出
	if teaconst.IsQuiting {
		return
	}

	var memoryStorage = this.memoryStorage
	if memoryStorage != nil {
		if item.Type == ItemTypeMemory {
			memoryStorage.AddToList(item)
			return
		}
	}

	item.MetaSize = SizeMeta + 128
	var hash = stringutil.Md5(item.Key)
	err := this.list.Add(hash, item)
	if err != nil && !strings.Contains(err.Error(), "UNIQUE constraint failed") {
		remotelogs.Error("CACHE", "add to list failed: "+err.Error())
	}
}

// Delete 删除某个键值对应的缓存
func (this *FileStorage) Delete(key string) error {
	// 是否正在退出
	if teaconst.IsQuiting {
		return nil
	}

	// 先尝试内存缓存
	this.runMemoryStorageSafety(func(memoryStorage *MemoryStorage) {
		_ = memoryStorage.Delete(key)
	})

	hash, path, _ := this.keyPath(key)
	err := this.list.Remove(hash)
	if err != nil {
		return err
	}
	err = this.removeCacheFile(path)
	if err == nil || os.IsNotExist(err) {
		return nil
	}

	return err
}

// Stat 统计
func (this *FileStorage) Stat() (*Stat, error) {
	return this.list.Stat(func(hash string) bool {
		return true
	})
}

// CleanAll 清除所有的缓存
func (this *FileStorage) CleanAll() error {
	this.locker.Lock()
	defer this.locker.Unlock()

	// 先尝试内存缓存
	this.runMemoryStorageSafety(func(memoryStorage *MemoryStorage) {
		_ = memoryStorage.CleanAll()
	})

	err := this.list.CleanAll()
	if err != nil {
		return err
	}

	// 删除缓存和目录
	// 不能直接删除子目录，比较危险

	var rootDirs = []string{this.options.Dir}
	var subDirs = this.subDirs // copy slice
	if len(subDirs) > 0 {
		for _, subDir := range subDirs {
			rootDirs = append(rootDirs, subDir.Path)
		}
	}

	var dirNameReg = regexp.MustCompile(`^[0-9a-f]{2}$`)
	for _, rootDir := range rootDirs {
		var dir = rootDir + "/p" + types.String(this.policy.Id)
		err = func(dir string) error {
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
				var subDir = info.Name()

				// 检查目录名
				if !dirNameReg.MatchString(subDir) {
					continue
				}

				// 修改目录名
				var tmpDir = dir + "/" + subDir + "." + timeutil.Format("YmdHis") + ".trash"
				err = os.Rename(dir+"/"+subDir, tmpDir)
				if err != nil {
					return err
				}
			}

			// 重新遍历待删除
			goman.New(func() {
				err = this.cleanDeletedDirs(dir)
				if err != nil {
					remotelogs.Warn("CACHE", "delete '*.trash' dirs failed: "+err.Error())
				} else {
					// try to clean again, to delete writing files when deleting
					time.Sleep(10 * time.Minute)
					_ = this.cleanDeletedDirs(dir)
				}
			})

			return nil
		}(dir)
		if err != nil {
			return err
		}
	}

	return nil
}

// Purge 清理过期的缓存
func (this *FileStorage) Purge(keys []string, urlType string) error {
	// 是否正在退出
	if teaconst.IsQuiting {
		return nil
	}

	// 先尝试内存缓存
	this.runMemoryStorageSafety(func(memoryStorage *MemoryStorage) {
		_ = memoryStorage.Purge(keys, urlType)
	})

	// 目录
	if urlType == "dir" {
		for _, key := range keys {
			// 检查是否有通配符 http(s)://*.example.com
			var schemeIndex = strings.Index(key, "://")
			if schemeIndex > 0 {
				var keyRight = key[schemeIndex+3:]
				if strings.HasPrefix(keyRight, "*.") {
					err := this.list.CleanMatchPrefix(key)
					if err != nil {
						return err
					}
					continue
				}
			}

			err := this.list.CleanPrefix(key)
			if err != nil {
				return err
			}
		}
		return nil
	}

	// URL
	for _, key := range keys {
		// 检查是否有通配符 http(s)://*.example.com
		var schemeIndex = strings.Index(key, "://")
		if schemeIndex > 0 {
			var keyRight = key[schemeIndex+3:]
			if strings.HasPrefix(keyRight, "*.") {
				err := this.list.CleanMatchKey(key)
				if err != nil {
					return err
				}
				continue
			}
		}

		// 普通的Key
		hash, path, _ := this.keyPath(key)
		err := this.removeCacheFile(path)
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
	this.runMemoryStorageSafety(func(memoryStorage *MemoryStorage) {
		memoryStorage.Stop()
	})

	if this.list != nil {
		_ = this.list.Reset()
	}

	if this.purgeTicker != nil {
		this.purgeTicker.Stop()
	}
	if this.hotTicker != nil {
		this.hotTicker.Stop()
	}

	if this.list != nil {
		_ = this.list.Close()
	}

	var openFileCache = this.openFileCache
	if openFileCache != nil {
		openFileCache.CloseAll()
	}

	this.ignoreKeys.Reset()
}

// TotalDiskSize 消耗的磁盘尺寸
func (this *FileStorage) TotalDiskSize() int64 {
	stat, err := fsutils.StatDeviceCache(this.options.Dir)
	if err == nil {
		return int64(stat.UsedSize())
	}
	return 0
}

// TotalMemorySize 内存尺寸
func (this *FileStorage) TotalMemorySize() int64 {
	var memoryStorage = this.memoryStorage
	if memoryStorage == nil {
		return 0
	}
	return memoryStorage.TotalMemorySize()
}

// IgnoreKey 忽略某个Key，即不缓存某个Key
func (this *FileStorage) IgnoreKey(key string, maxSize int64) {
	this.ignoreKeys.Push(types.String(maxSize) + "$" + key)
}

// CanSendfile 是否支持Sendfile
func (this *FileStorage) CanSendfile() bool {
	if this.options == nil {
		return false
	}
	return this.options.EnableSendfile
}

// 获取Key对应的文件路径
func (this *FileStorage) keyPath(key string) (hash string, path string, diskIsFull bool) {
	hash = stringutil.Md5(key)
	var dir string
	dir, diskIsFull = this.subDir(hash)
	path = dir + "/" + hash + ".cache"
	return
}

// 获取Hash对应的文件路径
func (this *FileStorage) hashPath(hash string) (path string, diskIsFull bool) {
	if len(hash) != HashKeyLength {
		return "", false
	}
	var dir string
	dir, diskIsFull = this.subDir(hash)
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
	this.initPurgeTicker()

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

	// 退出时停止
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

	return nil
}

// 清理任务
// TODO purge每个分区
func (this *FileStorage) purgeLoop() {
	// load
	systemLoad, _ := load.Avg()

	// TODO 计算平均最近每日新增用量

	// 计算是否应该开启LFU清理
	var capacityBytes = this.diskCapacityBytes()
	var startLFU = false
	var requireFullLFU = false // 是否需要完整执行LFU
	var lfuFreePercent = this.policy.PersistenceLFUFreePercent
	if lfuFreePercent <= 0 {
		lfuFreePercent = 5

		if systemLoad == nil || systemLoad.Load5 > 10 {
			// 2TB级别以上
			if capacityBytes>>30 > 2000 {
				lfuFreePercent = 100 /** GB **/ / float32(capacityBytes>>30) * 100 /** % **/
				if lfuFreePercent > 3 {
					lfuFreePercent = 3
				}
			}
		}
	}

	var hasFullDisk = this.hasFullDisk()
	if hasFullDisk {
		startLFU = true
	} else {
		var usedPercent = float32(this.TotalDiskSize()*100) / float32(capacityBytes)
		if capacityBytes > 0 {
			if lfuFreePercent < 100 {
				if usedPercent >= 100-lfuFreePercent {
					startLFU = true
				}
			}
		}
	}

	// 清理过期
	{
		var times = 1

		// 空闲时间多清理
		if systemLoad != nil {
			if systemLoad.Load5 < 3 {
				times = 5
			} else if systemLoad.Load5 < 5 {
				times = 3
			} else if systemLoad.Load5 < 10 {
				times = 2
			}
		}

		// 高速硬盘多清理
		if fsutils.DiskIsExtremelyFast() {
			times *= 8
		} else if fsutils.DiskIsFast() {
			times *= 4
		}

		// 处于LFU阈值时，多清理
		if startLFU {
			times *= 5
		}

		var purgeCount = this.policy.PersistenceAutoPurgeCount
		if purgeCount <= 0 {
			purgeCount = 1000

			if fsutils.DiskIsExtremelyFast() {
				purgeCount = 4000
			} else if fsutils.DiskIsFast() {
				purgeCount = 2000
			}
		}

		for i := 0; i < times; i++ {
			countFound, err := this.list.Purge(purgeCount, func(hash string) error {
				path, _ := this.hashPath(hash)
				fsutils.WriteBegin()
				err := this.removeCacheFile(path)
				fsutils.WriteEnd()
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
				if i == 0 && startLFU {
					requireFullLFU = true
				}

				break
			}

			time.Sleep(1 * time.Second)
		}
	}

	// 磁盘空间不足时，清除老旧的缓存
	if startLFU {
		var maxCount = 1000
		var maxLoops = 5

		if fsutils.DiskIsExtremelyFast() {
			maxCount = 4000
		} else if fsutils.DiskIsFast() {
			maxCount = 2000
		}

		var total, _ = this.list.Count()
		if total > 0 {
			for {
				maxLoops--
				if maxLoops <= 0 {
					break
				}

				// 开始清理
				var count = types.Int(math.Ceil(float64(total) * float64(lfuFreePercent*2) / 100))
				if count <= 0 {
					break
				}

				// 限制单次清理的条数，防止占用太多系统资源
				if count > maxCount {
					count = maxCount
				}

				var before = time.Now()
				err := this.list.PurgeLFU(count, func(hash string) error {
					path, _ := this.hashPath(hash)
					fsutils.WriteBegin()
					err := this.removeCacheFile(path)
					fsutils.WriteEnd()
					if err != nil && !os.IsNotExist(err) {
						remotelogs.Error("CACHE", "purge '"+path+"' error: "+err.Error())
					}

					return nil
				})

				var prefix = ""
				if requireFullLFU {
					prefix = "fully "
				}
				remotelogs.Println("CACHE", prefix+"LFU purge policy '"+this.policy.Name+"' id: "+types.String(this.policy.Id)+", count: "+types.String(count)+", cost: "+fmt.Sprintf("%.2fms", time.Since(before).Seconds()*1000))

				if err != nil {
					remotelogs.Warn("CACHE", "purge file storage in LFU failed: "+err.Error())
				}

				// 检查硬盘空间状态
				if !requireFullLFU && !this.hasFullDisk() {
					break
				}
			}
		}
	}
}

// 热点数据任务
func (this *FileStorage) hotLoop() {
	var memoryStorage = this.memoryStorage // copy
	if memoryStorage == nil {
		return
	}

	// check memory space size
	if !memoryStorage.HasFreeSpaceForHotItems() {
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
		if v.Hits <= 1 {
			continue
		}
		result = append(result, v)
	}

	this.hotMap = map[string]*HotItem{}
	this.hotMapLocker.Unlock()

	// 取Top10%写入内存
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

			// 如果即将过期，则忽略
			var nowUnixTime = time.Now().Unix()
			if reader.ExpiresAt() <= nowUnixTime+600 {
				continue
			}

			// 计算合适的过期时间
			var bestExpiresAt = nowUnixTime + HotItemLifeSeconds
			var hotTimes = int64(item.Hits) / 1000
			if hotTimes > 8 {
				hotTimes = 8
			}
			bestExpiresAt += hotTimes * HotItemLifeSeconds
			var expiresAt = reader.ExpiresAt()
			if expiresAt <= 0 || expiresAt > bestExpiresAt {
				expiresAt = bestExpiresAt
			}

			writer, err := memoryStorage.openWriter(item.Key, expiresAt, reader.Status(), types.Int(reader.HeaderSize()), reader.BodySize(), -1, false)
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
				goNext = true
				if n > 0 {
					_, err = writer.Write(buf[:n])
					if err != nil {
						goNext = false
					}
				}
				return
			})
			if err != nil {
				_ = reader.Close()
				_ = writer.Discard()
				continue
			}

			memoryStorage.AddToList(&Item{
				Type:       writer.ItemType(),
				Key:        item.Key,
				Host:       ParseHost(item.Key),
				ExpiredAt:  expiresAt,
				HeaderSize: writer.HeaderSize(),
				BodySize:   writer.BodySize(),
			})

			_ = reader.Close()
			_ = writer.Close()
		}
	}
}

func (this *FileStorage) diskCapacityBytes() int64 {
	var c1 = this.policy.CapacityBytes()
	var nodeCapacity = SharedManager.MaxDiskCapacity // copy
	if nodeCapacity != nil {
		var c2 = nodeCapacity.Bytes()
		if c2 > 0 {
			if this.mainDiskTotalSize > 0 && c2 >= int64(this.mainDiskTotalSize) {
				c2 = int64(this.mainDiskTotalSize) * 95 / 100 // keep 5% free
			}
			return c2
		}
	}

	if c1 <= 0 || (this.mainDiskTotalSize > 0 && c1 >= int64(this.mainDiskTotalSize)) {
		c1 = int64(this.mainDiskTotalSize) * 95 / 100 // keep 5% free
	}

	return c1
}

// remove all *.trash directories under policy directory
func (this *FileStorage) cleanAllDeletedDirs() {
	var rootDirs = []string{this.options.Dir}
	var subDirs = this.subDirs // copy slice
	if len(subDirs) > 0 {
		for _, subDir := range subDirs {
			rootDirs = append(rootDirs, subDir.Path)
		}
	}

	for _, rootDir := range rootDirs {
		var dir = rootDir + "/p" + types.String(this.policy.Id)
		goman.New(func() {
			_ = this.cleanDeletedDirs(dir)
		})
	}
}

// 清理 *.trash 目录
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
		var subDir = info.Name()
		if !strings.HasSuffix(subDir, ".trash") {
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
	if rands.Int(0, rate) == 0 {
		var memoryStorage = this.memoryStorage

		// 增加到热点
		// 这里不收录缓存尺寸过大的文件
		if memoryStorage != nil && reader.BodySize() > 0 && reader.BodySize() < 128*sizes.M {
			this.hotMapLocker.Lock()
			hotItem, ok := this.hotMap[key]

			if ok {
				hotItem.Hits++
			} else if len(this.hotMap) < HotItemSize { // 控制数量
				this.hotMap[key] = &HotItem{
					Key:  key,
					Hits: 1,
				}
			}
			this.hotMapLocker.Unlock()

			// 只有重复点击的才增加点击量
			if ok {
				var hitErr = this.list.IncreaseHit(hash)
				if hitErr != nil {
					// 此错误可以忽略
					remotelogs.Error("CACHE", "increase hit failed: "+hitErr.Error())
				}
			}
		}
	}
}

// 删除缓存文件
func (this *FileStorage) removeCacheFile(path string) error {
	var openFileCache = this.openFileCache
	if openFileCache != nil {
		openFileCache.Close(path)
	}

	var err = os.Remove(path)
	if err == nil || os.IsNotExist(err) {
		err = nil

		// 删除Partial相关
		var partialPath = PartialRangesFilePath(path)
		if openFileCache != nil {
			openFileCache.Close(partialPath)
		}

		_, statErr := os.Stat(partialPath)
		if statErr == nil {
			_ = os.Remove(partialPath)
		}
	}
	return err
}

// 创建当前策略包含的内存缓存
func (this *FileStorage) createMemoryStorage() error {
	var memoryPolicy = &serverconfigs.HTTPCachePolicy{
		Id:          this.policy.Id,
		IsOn:        this.policy.IsOn,
		Name:        this.policy.Name,
		Description: this.policy.Description,
		Capacity:    this.options.MemoryPolicy.Capacity,
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
	err := memoryPolicy.Init()
	if err != nil {
		return err
	}
	var memoryStorage = NewMemoryStorage(memoryPolicy, this)
	err = memoryStorage.Init()
	if err != nil {
		return err
	}
	this.memoryStorage = memoryStorage

	return nil
}

func (this *FileStorage) initPurgeTicker() {
	var autoPurgeInterval = this.policy.PersistenceAutoPurgeInterval
	if autoPurgeInterval <= 0 {
		autoPurgeInterval = 30
		if Tea.IsTesting() {
			autoPurgeInterval = 10
		}
	}
	if this.purgeTicker != nil {
		this.purgeTicker.Stop()
	}
	this.purgeTicker = utils.NewTicker(time.Duration(autoPurgeInterval) * time.Second)
	goman.New(func() {
		for this.purgeTicker.Next() {
			trackers.Run("FILE_CACHE_STORAGE_PURGE_LOOP", func() {
				this.purgeLoop()
			})
		}
	})
}

func (this *FileStorage) initOpenFileCache() {
	var err error

	var oldOpenFileCache = this.openFileCache

	// 启用新的
	if this.options.OpenFileCache != nil && this.options.OpenFileCache.IsOn && this.options.OpenFileCache.Max > 0 {
		this.openFileCache, err = NewOpenFileCache(this.options.OpenFileCache.Max)
		if err != nil {
			remotelogs.Error("CACHE", "open file cache failed: "+err.Error())
		}
	}

	// 关闭老的
	if oldOpenFileCache != nil {
		oldOpenFileCache.CloseAll()
	}
}

func (this *FileStorage) runMemoryStorageSafety(f func(memoryStorage *MemoryStorage)) {
	var memoryStorage = this.memoryStorage // copy
	if memoryStorage != nil {
		f(memoryStorage)
	}
}

// 检查磁盘剩余空间
func (this *FileStorage) checkDiskSpace() {
	var minFreeSize = DefaultMinDiskFreeSpace

	var options = this.options // copy
	if options != nil && options.MinFreeSize != nil && options.MinFreeSize.Bytes() > 0 {
		minFreeSize = uint64(options.MinFreeSize.Bytes())
	}

	if options != nil && len(options.Dir) > 0 {
		stat, err := fsutils.StatDevice(options.Dir)
		if err == nil {
			this.mainDiskIsFull = stat.FreeSize() < minFreeSize
			this.mainDiskTotalSize = stat.TotalSize()

			// check capacity (only on main directory) when node capacity had not been set
			if !this.mainDiskIsFull {
				var capacityBytes int64
				var maxDiskCapacity = SharedManager.MaxDiskCapacity // copy
				if maxDiskCapacity != nil && maxDiskCapacity.Bytes() > 0 {
					capacityBytes = SharedManager.MaxDiskCapacity.Bytes()
				} else {
					var policy = this.policy // copy
					if policy != nil {
						capacityBytes = policy.CapacityBytes() // copy
					}
				}

				if capacityBytes > 0 && stat.UsedSize() >= uint64(capacityBytes) {
					this.mainDiskIsFull = true
				}
			}
		}
	}
	var subDirs = this.subDirs // copy slice
	for _, subDir := range subDirs {
		stat, err := fsutils.StatDevice(subDir.Path)
		if err == nil {
			subDir.IsFull = stat.FreeSize() < minFreeSize
		}
	}
}

// 检查是否有已满的磁盘分区
func (this *FileStorage) hasFullDisk() bool {
	this.checkDiskSpace()

	var hasFullDisk = this.mainDiskIsFull
	if !hasFullDisk {
		var subDirs = this.subDirs // copy slice
		for _, subDir := range subDirs {
			if subDir.IsFull {
				hasFullDisk = true
				break
			}
		}
	}
	return hasFullDisk
}

// 获取目录
func (this *FileStorage) subDir(hash string) (dirPath string, dirIsFull bool) {
	var suffix = "/p" + types.String(this.policy.Id) + "/" + hash[:2] + "/" + hash[2:4]

	if len(hash) < 4 {
		return this.options.Dir + suffix, this.mainDiskIsFull
	}

	var subDirs = this.subDirs // copy slice
	var countSubDirs = len(subDirs)
	if countSubDirs == 0 {
		return this.options.Dir + suffix, this.mainDiskIsFull
	}

	countSubDirs++ // add main dir

	// 最多只支持16个目录
	if countSubDirs > 16 {
		countSubDirs = 16
	}

	var dirIndex = this.charCode(hash[0]) % uint8(countSubDirs)
	if dirIndex == 0 {
		return this.options.Dir + suffix, this.mainDiskIsFull
	}
	var subDir = subDirs[dirIndex-1]
	return subDir.Path + suffix, subDir.IsFull
}

// ScanGarbageCaches 清理目录中“失联”的缓存文件
// “失联”为不在HashMap中的文件
func (this *FileStorage) ScanGarbageCaches(fileCallback func(path string) error) error {
	if !this.list.(*FileList).HashMapIsLoaded() {
		return errors.New("cache list is loading")
	}

	var mainDir = this.options.Dir
	var allDirs = []string{mainDir}
	var subDirs = this.subDirs // copy
	for _, subDir := range subDirs {
		allDirs = append(allDirs, subDir.Path)
	}

	var countDirs = 0

	// process progress
	var progressSock = gosock.NewTmpSock(teaconst.CacheGarbageSockName)
	_, sockErr := progressSock.SendTimeout(&gosock.Command{Code: "progress", Params: map[string]any{"progress": 0}}, 1*time.Second)
	var canReportProgress = sockErr == nil
	var lastProgress float64
	var countFound = 0

	for _, subDir := range allDirs {
		var dir0 = subDir + "/p" + types.String(this.policy.Id)
		dir1Matches, err := filepath.Glob(dir0 + "/*")
		if err != nil {
			// ignore error
			continue
		}

		for _, dir1 := range dir1Matches {
			if len(filepath.Base(dir1)) != 2 {
				continue
			}

			dir2Matches, err := filepath.Glob(dir1 + "/*")
			if err != nil {
				// ignore error
				continue
			}
			for _, dir2 := range dir2Matches {
				if len(filepath.Base(dir2)) != 2 {
					continue
				}

				countDirs++

				// report progress
				if canReportProgress {
					var progress = float64(countDirs) / 65536
					if fmt.Sprintf("%.2f", lastProgress) != fmt.Sprintf("%.2f", progress) {
						lastProgress = progress
						_, _ = progressSock.SendTimeout(&gosock.Command{Code: "progress", Params: map[string]any{
							"progress": progress,
							"count":    countFound,
						}}, 100*time.Millisecond)
					}
				}

				fileMatches, err := filepath.Glob(dir2 + "/*.cache")
				if err != nil {
					// ignore error
					continue
				}

				for _, file := range fileMatches {
					var filename = filepath.Base(file)
					var hash = strings.TrimSuffix(filename, ".cache")
					if len(hash) != HashKeyLength {
						continue
					}

					isReady, found := this.list.(*FileList).ExistQuick(hash)
					if !isReady {
						continue
					}

					if found {
						continue
					}

					// 检查文件正在被写入
					stat, err := os.Stat(file)
					if err != nil {
						continue
					}
					if fasttime.Now().Unix()-stat.ModTime().Unix() < 300 /** 5 minutes **/ {
						continue
					}

					if fileCallback != nil {
						countFound++
						err = fileCallback(file)
						if err != nil {
							return err
						}
					}
				}
			}
		}
	}

	// 100% progress
	if canReportProgress && lastProgress != 1 {
		_, _ = progressSock.SendTimeout(&gosock.Command{Code: "progress", Params: map[string]any{
			"progress": 1,
			"count":    countFound,
		}}, 100*time.Millisecond)
	}

	return nil
}

// 计算字节数字代号
func (this *FileStorage) charCode(r byte) uint8 {
	if r >= '0' && r <= '9' {
		return r - '0'
	}
	if r >= 'a' && r <= 'z' {
		return r - 'a' + 10
	}
	return 0
}
