package caches

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs"
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs/shared"
	"github.com/TeaOSLab/EdgeNode/internal/events"
	"github.com/TeaOSLab/EdgeNode/internal/remotelogs"
	"github.com/TeaOSLab/EdgeNode/internal/utils"
	"github.com/iwind/TeaGo/Tea"
	stringutil "github.com/iwind/TeaGo/utils/string"
	"io"
	"os"
	"path/filepath"
	"regexp"
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

var (
	ErrNotFound      = errors.New("cache not found")
	ErrFileIsWriting = errors.New("the file is writing")
	ErrInvalidRange  = errors.New("invalid range")
)

// 文件缓存
//   文件结构：
//    [expires time] | [ status ] | [url length] | [header length] | [body length] | [url] [header data] [body data]
type FileStorage struct {
	policy        *serverconfigs.HTTPCachePolicy
	cacheConfig   *serverconfigs.HTTPFileCacheStorage // 二级缓存
	memoryStorage *MemoryStorage                      // 一级缓存
	totalSize     int64

	list   *List
	locker sync.RWMutex
	ticker *utils.Ticker
}

func NewFileStorage(policy *serverconfigs.HTTPCachePolicy) *FileStorage {
	return &FileStorage{
		policy: policy,
		list:   NewList(),
	}
}

// 获取当前的Policy
func (this *FileStorage) Policy() *serverconfigs.HTTPCachePolicy {
	return this.policy
}

// 初始化
func (this *FileStorage) Init() error {
	this.list.OnAdd(func(item *Item) {
		atomic.AddInt64(&this.totalSize, item.TotalSize())
	})
	this.list.OnRemove(func(item *Item) {
		atomic.AddInt64(&this.totalSize, -item.TotalSize())
	})

	this.locker.Lock()
	defer this.locker.Unlock()

	before := time.Now()
	cacheDir := ""
	defer func() {
		// 统计
		count := 0
		size := int64(0)
		if this.list != nil {
			stat := this.list.Stat(func(hash string) bool {
				return true
			})
			count = stat.Count
			size = stat.Size
		}

		cost := time.Since(before).Seconds() * 1000
		sizeMB := strconv.FormatInt(size, 10) + " Bytes"
		if size > 1024*1024*1024 {
			sizeMB = fmt.Sprintf("%.3f G", float64(size)/1024/1024/1024)
		} else if size > 1024*1024 {
			sizeMB = fmt.Sprintf("%.3f M", float64(size)/1024/1024)
		} else if size > 1024 {
			sizeMB = fmt.Sprintf("%.3f K", float64(size)/1024)
		}
		remotelogs.Println("CACHE", "init policy "+strconv.FormatInt(this.policy.Id, 10)+" from '"+cacheDir+"', cost: "+fmt.Sprintf("%.2f", cost)+" ms, count: "+strconv.Itoa(count)+", size: "+sizeMB)
	}()

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
	cacheDir = cacheConfig.Dir

	if !filepath.IsAbs(this.cacheConfig.Dir) {
		this.cacheConfig.Dir = Tea.Root + Tea.DS + this.cacheConfig.Dir
	}

	dir := this.cacheConfig.Dir

	if len(dir) == 0 {
		return errors.New("[CACHE]cache storage dir can not be empty")
	}

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

	// 初始化list
	err = this.initList()
	if err != nil {
		return err
	}

	// 加载内存缓存
	if this.cacheConfig.MemoryPolicy != nil {
		memoryCapacity := this.cacheConfig.MemoryPolicy.Capacity
		if memoryCapacity != nil && memoryCapacity.Count > 0 {
			memoryPolicy := &serverconfigs.HTTPCachePolicy{
				Id:          this.policy.Id,
				IsOn:        this.policy.IsOn,
				Name:        this.policy.Name,
				Description: this.policy.Description,
				Capacity:    memoryCapacity,
				MaxKeys:     this.policy.MaxKeys,
				MaxSize:     &shared.SizeCapacity{Count: 128, Unit: shared.SizeCapacityUnitMB}, // TODO 将来可以修改
				Type:        serverconfigs.CachePolicyStorageMemory,
				Options:     this.policy.Options,
				Life:        this.policy.Life,
				MinLife:     this.policy.MinLife,
				MaxLife:     this.policy.MaxLife,
			}
			err = memoryPolicy.Init()
			if err != nil {
				return err
			}
			memoryStorage := NewMemoryStorage(memoryPolicy)
			err = memoryStorage.Init()
			if err != nil {
				return err
			}
			this.memoryStorage = memoryStorage
		}
	}

	return nil
}

func (this *FileStorage) OpenReader(key string) (Reader, error) {
	// 先尝试内存缓存
	if this.memoryStorage != nil {
		reader, err := this.memoryStorage.OpenReader(key)
		if err == nil {
			return reader, err
		}
	}

	hash, path := this.keyPath(key)
	if !this.list.Exist(hash) {
		return nil, ErrNotFound
	}

	// TODO 尝试使用mmap加快读取速度
	fp, err := os.OpenFile(path, os.O_RDONLY, 0444)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
		return nil, ErrNotFound
	}

	reader := NewFileReader(fp)
	if err != nil {
		return nil, err
	}
	err = reader.Init()
	if err != nil {
		return nil, err
	}
	return reader, nil
}

// 打开缓存文件等待写入
func (this *FileStorage) OpenWriter(key string, expiredAt int64, status int) (Writer, error) {
	// 先尝试内存缓存
	if this.memoryStorage != nil {
		writer, err := this.memoryStorage.OpenWriter(key, expiredAt, status)
		if err == nil {
			return writer, nil
		}
	}

	// 检查是否超出最大值
	if this.policy.MaxKeys > 0 && this.list.Count() > this.policy.MaxKeys {
		return nil, errors.New("write file cache failed: too many keys in cache storage")
	}
	if this.policy.CapacityBytes() > 0 && this.policy.CapacityBytes() <= this.totalSize {
		return nil, errors.New("write file cache failed: over disk size, real size: " + strconv.FormatInt(this.totalSize, 10) + " bytes")
	}

	hash := stringutil.Md5(key)
	dir := this.cacheConfig.Dir + "/p" + strconv.FormatInt(this.policy.Id, 10) + "/" + hash[:2] + "/" + hash[2:4]
	_, err := os.Stat(dir)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
		err = os.MkdirAll(dir, 0777)
		if err != nil {
			return nil, err
		}
	}

	// 先删除
	this.list.Remove(hash)

	path := dir + "/" + hash + ".cache.tmp"
	writer, err := os.OpenFile(path, os.O_CREATE|os.O_SYNC|os.O_WRONLY, 0666)
	if err != nil {
		return nil, err
	}

	isOk := false
	removeOnFailure := true
	defer func() {
		if err != nil {
			isOk = false
		}

		// 如果出错了，就删除文件，避免写一半
		if !isOk {
			_ = writer.Close()
			if removeOnFailure {
				_ = os.Remove(path)
			}
		}
	}()

	// 尝试锁定，如果锁定失败，则直接返回
	err = syscall.Flock(int(writer.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
	if err != nil {
		removeOnFailure = false
		return nil, ErrFileIsWriting
	}

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

	isOk = true

	return NewFileWriter(writer, key, expiredAt), nil
}

// 添加到List
func (this *FileStorage) AddToList(item *Item) {
	if this.memoryStorage != nil {
		if item.Type == ItemTypeMemory {
			this.memoryStorage.AddToList(item)
			return
		}
	}

	item.MetaSize = SizeMeta
	hash := stringutil.Md5(item.Key)
	this.list.Add(hash, item)
}

// 删除某个键值对应的缓存
func (this *FileStorage) Delete(key string) error {
	this.locker.Lock()
	defer this.locker.Unlock()

	// 先尝试内存缓存
	if this.memoryStorage != nil {
		_ = this.memoryStorage.Delete(key)
	}

	hash, path := this.keyPath(key)
	this.list.Remove(hash)
	err := os.Remove(path)
	if err == nil || os.IsNotExist(err) {
		return nil
	}
	return err
}

// 统计
func (this *FileStorage) Stat() (*Stat, error) {
	this.locker.RLock()
	defer this.locker.RUnlock()

	return this.list.Stat(func(hash string) bool {
		return true
	}), nil
}

// 清除所有的缓存
func (this *FileStorage) CleanAll() error {
	this.locker.Lock()
	defer this.locker.Unlock()

	// 先尝试内存缓存
	if this.memoryStorage != nil {
		_ = this.memoryStorage.CleanAll()
	}

	this.list.Reset()

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

		// 删除目录
		err = os.RemoveAll(dir + "/" + subDir)
		if err != nil {
			return err
		}
	}

	return nil
}

// 清理过期的缓存
func (this *FileStorage) Purge(keys []string, urlType string) error {
	this.locker.Lock()
	defer this.locker.Unlock()

	// 先尝试内存缓存
	if this.memoryStorage != nil {
		_ = this.memoryStorage.Purge(keys, urlType)
	}

	// 目录
	if urlType == "dir" {
		resultKeys := []string{}
		for _, key := range keys {
			resultKeys = append(resultKeys, this.list.FindKeysWithPrefix(key)...)
		}
		keys = resultKeys
	}

	// 文件
	for _, key := range keys {
		hash, path := this.keyPath(key)
		if !this.list.Exist(hash) {
			err := os.Remove(path)
			if err != nil && !os.IsNotExist(err) {
				return err
			}
			continue
		}

		err := os.Remove(path)
		if err != nil && !os.IsNotExist(err) {
			return err
		}
		this.list.Remove(hash)
	}
	return nil
}

// 停止
func (this *FileStorage) Stop() {
	this.locker.Lock()
	defer this.locker.Unlock()

	// 先尝试内存缓存
	if this.memoryStorage != nil {
		this.memoryStorage.Stop()
	}

	this.list.Reset()
	if this.ticker != nil {
		this.ticker.Stop()
	}
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
	this.list.Reset()

	dir := this.dir()

	// 清除tmp
	files, err := filepath.Glob(dir + "/*/*/*.cache.tmp")
	if err != nil {
		return err
	}
	for _, path := range files {
		_ = os.Remove(path)
	}

	// 加载缓存
	files, err = filepath.Glob(dir + "/*/*/*.cache")
	if err != nil {
		return err
	}
	for _, path := range files {
		basename := filepath.Base(path)
		index := strings.LastIndex(basename, ".")
		if index < 0 {
			continue
		}
		hash := basename[:index]

		// 解析文件信息
		item, err := this.decodeFile(path)
		if err != nil {
			if err != ErrNotFound {
				remotelogs.Error("CACHE", "decode path '"+path+"': "+err.Error())
			}
			continue
		}
		if item == nil {
			continue
		}
		this.list.Add(hash, item)
	}

	// 启动定时清理任务
	this.ticker = utils.NewTicker(30 * time.Second)
	events.On(events.EventQuit, func() {
		remotelogs.Println("CACHE", "quit clean timer")
		var ticker = this.ticker
		if ticker != nil {
			ticker.Stop()
		}
	})
	go func() {
		for this.ticker.Next() {
			this.purgeLoop()
		}
	}()

	return nil
}

// 解析文件信息
func (this *FileStorage) decodeFile(path string) (*Item, error) {
	fp, err := os.OpenFile(path, os.O_RDONLY, 0444)
	if err != nil {
		return nil, err
	}

	isAllOk := false
	defer func() {
		_ = fp.Close()

		if !isAllOk {
			_ = os.Remove(path)
		}
	}()

	item := &Item{
		Type:     ItemTypeFile,
		MetaSize: SizeMeta,
	}

	bytes4 := make([]byte, 4)

	// 过期时间
	ok, err := this.readToBuff(fp, bytes4)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrNotFound
	}
	item.ExpiredAt = int64(binary.BigEndian.Uint32(bytes4))

	// 是否已过期
	if item.ExpiredAt < time.Now().Unix() {
		return nil, ErrNotFound
	}

	// URL Size
	_, err = fp.Seek(int64(SizeExpiresAt+SizeStatus), io.SeekStart)
	if err != nil {
		return nil, err
	}
	ok, err = this.readToBuff(fp, bytes4)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrNotFound
	}
	urlSize := binary.BigEndian.Uint32(bytes4)

	// Header Size
	ok, err = this.readToBuff(fp, bytes4)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrNotFound
	}
	item.HeaderSize = int64(binary.BigEndian.Uint32(bytes4))

	// Body Size
	bytes8 := make([]byte, 8)
	ok, err = this.readToBuff(fp, bytes8)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrNotFound
	}
	item.BodySize = int64(binary.BigEndian.Uint64(bytes8))

	// URL
	if urlSize > 0 {
		data := utils.BytePool1024.Get()
		result, ok, err := this.readN(fp, data, int(urlSize))
		utils.BytePool1024.Put(data)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, ErrNotFound
		}
		item.Key = string(result)
	}

	isAllOk = true

	return item, nil
}

// 清理任务
func (this *FileStorage) purgeLoop() {
	this.list.Purge(1000, func(hash string) {
		path := this.hashPath(hash)
		err := os.Remove(path)
		if err != nil && !os.IsNotExist(err) {
			remotelogs.Error("CACHE", "purge '"+path+"' error: "+err.Error())
		}
	})
}

func (this *FileStorage) readToBuff(fp *os.File, buf []byte) (ok bool, err error) {
	n, err := fp.Read(buf)
	if err != nil {
		return false, err
	}
	ok = n == len(buf)
	return
}

func (this *FileStorage) readN(fp *os.File, buf []byte, total int) (result []byte, ok bool, err error) {
	for {
		n, err := fp.Read(buf)
		if err != nil {
			return nil, false, err
		}
		if n > 0 {
			if n >= total {
				result = append(result, buf[:total]...)
				ok = true
				return result, ok, nil
			} else {
				total -= n
				result = append(result, buf[:n]...)
			}
		}
	}
}
