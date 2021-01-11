package caches

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs"
	"github.com/TeaOSLab/EdgeNode/internal/events"
	"github.com/TeaOSLab/EdgeNode/internal/remotelogs"
	"github.com/TeaOSLab/EdgeNode/internal/utils"
	"github.com/iwind/TeaGo/Tea"
	"github.com/iwind/TeaGo/types"
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
	SizeExpiredAt = 10
	SizeKeyLength = 4
	SizeNL        = 1
	SizeEnd       = 4
)

var (
	ErrNotFound      = errors.New("cache not found")
	ErrFileIsWriting = errors.New("the file is writing")
)

type FileStorage struct {
	policy      *serverconfigs.HTTPCachePolicy
	cacheConfig *serverconfigs.HTTPFileCacheStorage
	totalSize   int64

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
		atomic.AddInt64(&this.totalSize, item.Size)
	})
	this.list.OnRemove(func(item *Item) {
		atomic.AddInt64(&this.totalSize, -item.Size)
	})

	this.locker.Lock()
	defer this.locker.Unlock()

	before := time.Now()
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
		remotelogs.Println("CACHE", "init policy "+strconv.FormatInt(this.policy.Id, 10)+", cost: "+fmt.Sprintf("%.2f", cost)+" ms, count: "+strconv.Itoa(count)+", size: "+fmt.Sprintf("%.3f", float64(size)/1024/1024)+" M")
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

	return nil
}

func (this *FileStorage) Read(key string, readerBuf []byte, callback func(data []byte, size int64, expiredAt int64, isEOF bool)) error {
	hash, path := this.keyPath(key)
	if !this.list.Exist(hash) {
		return ErrNotFound
	}

	this.locker.RLock()
	defer this.locker.RUnlock()

	// TODO 尝试使用mmap加快读取速度
	fp, err := os.OpenFile(path, os.O_RDONLY, 0444)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		return ErrNotFound
	}
	defer func() {
		_ = fp.Close()
	}()

	// 是否过期
	buf := make([]byte, SizeExpiredAt)
	n, err := fp.Read(buf)
	if err != nil {
		return err
	}
	if n != len(buf) {
		return ErrNotFound
	}

	expiredAt := types.Int64(string(buf))
	if expiredAt < time.Now().Unix() {
		// 已过期
		_ = fp.Close()
		_ = os.Remove(path)

		return ErrNotFound
	}

	buf = make([]byte, SizeKeyLength)
	n, err = fp.Read(buf)
	if err != nil {
		return err
	}
	if n != len(buf) {
		return ErrNotFound
	}
	keyLength := int(binary.BigEndian.Uint32(buf))

	offset, err := fp.Seek(-SizeEnd, io.SeekEnd)
	if err != nil {
		return err
	}
	buf = make([]byte, SizeEnd)
	n, err = fp.Read(buf)
	if n != len(buf) {
		return ErrNotFound
	}
	if string(buf) != "\n$$$" {
		_ = fp.Close()
		_ = os.Remove(path)
		return ErrNotFound
	}
	startOffset := SizeExpiredAt + SizeKeyLength + keyLength + SizeNL
	size := int(offset) + SizeEnd - startOffset
	valueSize := offset - int64(startOffset)

	_, err = fp.Seek(int64(startOffset), io.SeekStart)
	if err != nil {
		return err
	}

	for {
		n, err := fp.Read(readerBuf)
		if n > 0 {
			size -= n
			if size < SizeEnd { // 已经到了末尾区域
				if n <= SizeEnd-size { // 已经到了末尾
					break
				} else {
					callback(readerBuf[:n-(SizeEnd-size)], valueSize, expiredAt, true)
				}
			} else {
				callback(readerBuf[:n], valueSize, expiredAt, false)
			}
		}
		if err != nil {
			if err != io.EOF {
				return err
			}

			break
		}
	}

	return nil
}

// 打开缓存文件等待写入
func (this *FileStorage) Open(key string, expiredAt int64) (Writer, error) {
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

	path := dir + "/" + hash + ".cache"
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
	_, err = writer.WriteString(fmt.Sprintf("%d", expiredAt))
	if err != nil {
		return nil, err
	}

	// 写入key length
	b := make([]byte, SizeKeyLength)
	binary.BigEndian.PutUint32(b, uint32(len(key)))
	_, err = writer.Write(b)
	if err != nil {
		return nil, err
	}

	// 写入key
	_, err = writer.WriteString(key + "\n")
	if err != nil {
		return nil, err
	}

	isOk = true

	return NewFileWriter(writer, key, expiredAt), nil
}

// 写入缓存数据
// 目录结构：$root/p$policyId/$hash[:2]/$hash[2:4]/$hash.cache
// 数据结构： [expiredAt] [key length] [key] \n value \n $$$
func (this *FileStorage) Write(key string, expiredAt int64, valueReader io.Reader) error {
	this.locker.Lock()
	defer this.locker.Unlock()

	hash := stringutil.Md5(key)
	dir := this.cacheConfig.Dir + "/p" + strconv.FormatInt(this.policy.Id, 10) + "/" + hash[:2] + "/" + hash[2:4]

	_, err := os.Stat(dir)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		err = os.MkdirAll(dir, 0777)
		if err != nil {
			return err
		}
	}
	path := dir + "/" + hash + ".cache"
	writer, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_SYNC|os.O_WRONLY, 0777)
	if err != nil {
		return err
	}

	isOk := false
	defer func() {
		err = writer.Close()
		if err != nil {
			isOk = false
		}

		// 如果出错了，就删除文件，避免写一半
		if !isOk {
			_ = os.Remove(path)
		}
	}()

	// 写入过期时间
	_, err = writer.WriteString(fmt.Sprintf("%d", expiredAt))
	if err != nil {
		return err
	}

	// 写入key length
	b := make([]byte, SizeKeyLength)
	binary.BigEndian.PutUint32(b, uint32(len(key)))
	_, err = writer.Write(b)
	if err != nil {
		return err
	}

	// 写入key
	_, err = writer.WriteString(key + "\n")
	if err != nil {
		return err
	}

	// 写入数据
	valueSize, err := io.Copy(writer, valueReader)
	if err != nil {
		return err
	}

	// 写入结束符
	_, err = writer.WriteString("\n$$$")

	isOk = true

	// 写入List
	this.list.Add(hash, &Item{
		Key:       key,
		ExpiredAt: expiredAt,
		ValueSize: valueSize,
		Size:      valueSize + SizeExpiredAt + SizeKeyLength + int64(len(key)) + SizeNL + SizeEnd,
	})

	return nil
}

// 添加到List
func (this *FileStorage) AddToList(item *Item) {
	item.Size = item.ValueSize + SizeExpiredAt + SizeKeyLength + int64(len(item.Key)) + SizeNL + SizeEnd
	hash := stringutil.Md5(item.Key)
	this.list.Add(hash, item)
}

// 删除某个键值对应的缓存
func (this *FileStorage) Delete(key string) error {
	this.locker.Lock()
	defer this.locker.Unlock()

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
	files, err := filepath.Glob(dir + "/*/*/*.cache")
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
	defer func() {
		_ = fp.Close()
	}()

	buf := make([]byte, SizeExpiredAt)
	n, err := fp.Read(buf)
	if err != nil {
		return nil, err
	}
	if n != len(buf) {
		// 数据格式错误
		_ = fp.Close()
		_ = os.Remove(path)

		return nil, ErrNotFound
	}
	expiredAt := types.Int64(string(buf))
	if expiredAt < time.Now().Unix() {
		// 已过期
		_ = fp.Close()
		_ = os.Remove(path)
		return nil, ErrNotFound
	}

	buf = make([]byte, SizeKeyLength)
	n, err = fp.Read(buf)
	if err != nil {
		return nil, err
	}
	keyLength := binary.BigEndian.Uint32(buf)

	buf = make([]byte, keyLength)
	n, err = fp.Read(buf)
	if err != nil {
		return nil, err
	}
	if n != int(keyLength) {
		// 数据格式错误
		_ = fp.Close()
		_ = os.Remove(path)
		return nil, ErrNotFound
	}

	stat, err := fp.Stat()
	if err != nil {
		return nil, err
	}

	item := &Item{}
	item.ExpiredAt = expiredAt
	item.Key = string(buf)
	item.Size = stat.Size()
	item.ValueSize = item.Size - SizeExpiredAt - SizeKeyLength - int64(keyLength) - SizeNL - SizeEnd
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
