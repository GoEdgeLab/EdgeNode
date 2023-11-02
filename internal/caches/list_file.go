// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package caches

import (
	"database/sql"
	"errors"
	"github.com/TeaOSLab/EdgeNode/internal/goman"
	"github.com/TeaOSLab/EdgeNode/internal/remotelogs"
	"github.com/TeaOSLab/EdgeNode/internal/ttlcache"
	"github.com/TeaOSLab/EdgeNode/internal/utils/dbs"
	"github.com/TeaOSLab/EdgeNode/internal/utils/fasttime"
	"github.com/TeaOSLab/EdgeNode/internal/utils/fnv"
	"github.com/TeaOSLab/EdgeNode/internal/zero"
	"github.com/iwind/TeaGo/types"
	"os"
	"strings"
	"sync"
	"time"
)

const CountFileDB = 20

// FileList 文件缓存列表管理
type FileList struct {
	dir    string
	dbList [CountFileDB]*FileListDB

	onAdd    func(item *Item)
	onRemove func(item *Item)

	memoryCache *ttlcache.Cache[zero.Zero]

	// 老数据库地址
	oldDir string
}

func NewFileList(dir string) ListInterface {
	return &FileList{
		dir:         dir,
		memoryCache: ttlcache.NewCache[zero.Zero](),
	}
}

func (this *FileList) SetOldDir(oldDir string) {
	this.oldDir = oldDir
}

func (this *FileList) Init() error {
	// 检查目录是否存在
	_, err := os.Stat(this.dir)
	if err != nil {
		err = os.MkdirAll(this.dir, 0777)
		if err != nil {
			return err
		}
		remotelogs.Println("CACHE", "create cache dir '"+this.dir+"'")
	}

	var dir = this.dir
	if dir == "/" {
		// 防止sqlite提示authority错误
		dir = ""
	}

	remotelogs.Println("CACHE", "loading database from '"+dir+"' ...")
	var wg = &sync.WaitGroup{}
	var locker = sync.Mutex{}
	var lastErr error

	for i := 0; i < CountFileDB; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()

			var db = NewFileListDB()
			dbErr := db.Open(dir + "/db-" + types.String(i) + ".db")
			if dbErr != nil {
				lastErr = dbErr
				return
			}

			dbErr = db.Init()
			if dbErr != nil {
				lastErr = dbErr
				return
			}

			locker.Lock()
			this.dbList[i] = db
			locker.Unlock()
		}(i)
	}
	wg.Wait()

	if lastErr != nil {
		return lastErr
	}

	// 升级老版本数据库
	goman.New(func() {
		this.upgradeOldDB()
	})

	return nil
}

func (this *FileList) Reset() error {
	// 不做任何事情
	return nil
}

func (this *FileList) Add(hash string, item *Item) error {
	var db = this.GetDB(hash)

	if !db.IsReady() {
		return nil
	}

	err := db.AddSync(hash, item)
	if err != nil {
		return err
	}

	this.memoryCache.Write(hash, zero.Zero{}, this.maxExpiresAtForMemoryCache(item.ExpiredAt))

	if this.onAdd != nil {
		this.onAdd(item)
	}
	return nil
}

func (this *FileList) Exist(hash string) (bool, error) {
	var db = this.GetDB(hash)

	if !db.IsReady() {
		return false, nil
	}

	// 如果Hash列表里不存在，那么必然不存在
	if !db.hashMap.Exist(hash) {
		return false, nil
	}

	var item = this.memoryCache.Read(hash)
	if item != nil {
		return true, nil
	}

	var row = db.existsByHashStmt.QueryRow(hash, time.Now().Unix())
	if row.Err() != nil {
		return false, nil
	}

	var expiredAt int64
	err := row.Scan(&expiredAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			err = nil
		}
		return false, err
	}
	this.memoryCache.Write(hash, zero.Zero{}, this.maxExpiresAtForMemoryCache(expiredAt))
	return true, nil
}

func (this *FileList) ExistQuick(hash string) (isReady bool, found bool) {
	var db = this.GetDB(hash)

	if !db.IsReady() || !db.HashMapIsLoaded() {
		return
	}

	isReady = true
	found = db.hashMap.Exist(hash)
	return
}

// CleanPrefix 清理某个前缀的缓存数据
func (this *FileList) CleanPrefix(prefix string) error {
	if len(prefix) == 0 {
		return nil
	}

	defer func() {
		// TODO 需要优化
		this.memoryCache.Clean()
	}()

	for _, db := range this.dbList {
		err := db.CleanPrefix(prefix)
		if err != nil {
			return err
		}
	}
	return nil
}

// CleanMatchKey 清理通配符匹配的缓存数据，类似于 https://*.example.com/hello
func (this *FileList) CleanMatchKey(key string) error {
	if len(key) == 0 {
		return nil
	}

	defer func() {
		// TODO 需要优化
		this.memoryCache.Clean()
	}()

	for _, db := range this.dbList {
		err := db.CleanMatchKey(key)
		if err != nil {
			return err
		}
	}
	return nil
}

// CleanMatchPrefix 清理通配符匹配的缓存数据，类似于 https://*.example.com/prefix/
func (this *FileList) CleanMatchPrefix(prefix string) error {
	if len(prefix) == 0 {
		return nil
	}

	defer func() {
		// TODO 需要优化
		this.memoryCache.Clean()
	}()

	for _, db := range this.dbList {
		err := db.CleanMatchPrefix(prefix)
		if err != nil {
			return err
		}
	}
	return nil
}

func (this *FileList) Remove(hash string) error {
	_, err := this.remove(hash, false)
	return err
}

// Purge 清理过期的缓存
// count 每次遍历的最大数量，控制此数字可以保证每次清理的时候不用花太多时间
// callback 每次发现过期key的调用
func (this *FileList) Purge(count int, callback func(hash string) error) (int, error) {
	count /= CountFileDB
	if count <= 0 {
		count = 100
	}

	var countFound = 0
	for _, db := range this.dbList {
		hashStrings, err := db.ListExpiredItems(count)
		if err != nil {
			return 0, nil
		}

		if len(hashStrings) == 0 {
			continue
		}

		countFound += len(hashStrings)

		// 不在 rows.Next() 循环中操作是为了避免死锁
		for _, hash := range hashStrings {
			_, err = this.remove(hash, true)
			if err != nil {
				return 0, err
			}

			err = callback(hash)
			if err != nil {
				return 0, err
			}
		}

		_, err = db.writeDB.Exec(`DELETE FROM "cacheItems" WHERE "hash" IN ('` + strings.Join(hashStrings, "', '") + `')`)
		if err != nil {
			return 0, err
		}
	}

	return countFound, nil
}

func (this *FileList) PurgeLFU(count int, callback func(hash string) error) error {
	count /= CountFileDB
	if count <= 0 {
		count = 100
	}

	for _, db := range this.dbList {
		hashStrings, err := db.ListLFUItems(count)
		if err != nil {
			return err
		}

		if len(hashStrings) == 0 {
			continue
		}

		// 不在 rows.Next() 循环中操作是为了避免死锁
		for _, hash := range hashStrings {
			_, err = this.remove(hash, true)
			if err != nil {
				return err
			}

			err = callback(hash)
			if err != nil {
				return err
			}
		}

		_, err = db.writeDB.Exec(`DELETE FROM "cacheItems" WHERE "hash" IN ('` + strings.Join(hashStrings, "', '") + `')`)
		if err != nil {
			return err
		}
	}
	return nil
}

func (this *FileList) CleanAll() error {
	defer this.memoryCache.Clean()

	for _, db := range this.dbList {
		err := db.CleanAll()
		if err != nil {
			return err
		}
	}

	return nil
}

func (this *FileList) Stat(check func(hash string) bool) (*Stat, error) {
	var result = &Stat{}

	for _, db := range this.dbList {
		if !db.IsReady() {
			return &Stat{}, nil
		}

		// 这里不设置过期时间、不使用 check 函数，目的是让查询更快速一些
		_ = check

		var row = db.statStmt.QueryRow()
		if row.Err() != nil {
			return nil, row.Err()
		}
		var stat = &Stat{}
		err := row.Scan(&stat.Count, &stat.Size, &stat.ValueSize)
		if err != nil {
			return nil, err
		}
		result.Count += stat.Count
		result.Size += stat.Size
		result.ValueSize += stat.ValueSize
	}

	return result, nil
}

// Count 总数量
// 常用的方法，所以避免直接查询数据库
func (this *FileList) Count() (int64, error) {
	var total int64
	for _, db := range this.dbList {
		count, err := db.Total()
		if err != nil {
			return 0, err
		}
		total += count
	}
	return total, nil
}

// IncreaseHit 增加点击量
func (this *FileList) IncreaseHit(hash string) error {
	var db = this.GetDB(hash)

	if !db.IsReady() {
		return nil
	}

	return db.IncreaseHitAsync(hash)
}

// OnAdd 添加事件
func (this *FileList) OnAdd(f func(item *Item)) {
	this.onAdd = f
}

// OnRemove 删除事件
func (this *FileList) OnRemove(f func(item *Item)) {
	this.onRemove = f
}

func (this *FileList) Close() error {
	this.memoryCache.Destroy()

	for _, db := range this.dbList {
		if db != nil {
			_ = db.Close()
		}
	}

	return nil
}

func (this *FileList) GetDBIndex(hash string) uint64 {
	return fnv.HashString(hash) % CountFileDB
}

func (this *FileList) GetDB(hash string) *FileListDB {
	return this.dbList[fnv.HashString(hash)%CountFileDB]
}

func (this *FileList) HashMapIsLoaded() bool {
	for _, db := range this.dbList {
		if !db.HashMapIsLoaded() {
			return false
		}
	}
	return true
}

func (this *FileList) remove(hash string, isDeleted bool) (notFound bool, err error) {
	var db = this.GetDB(hash)

	if !db.IsReady() {
		return false, nil
	}

	// HashMap中不存在，则确定不存在
	if !db.hashMap.Exist(hash) {
		return true, nil
	}
	defer db.hashMap.Delete(hash)

	// 从缓存中删除
	this.memoryCache.Delete(hash)

	if !isDeleted {
		err = db.DeleteSync(hash)
		if err != nil {
			return false, db.WrapError(err)
		}
	}

	if this.onRemove != nil {
		// when remove file item, no any extra information needed
		this.onRemove(nil)
	}

	return false, nil
}

// 升级老版本数据库
func (this *FileList) upgradeOldDB() {
	if len(this.oldDir) == 0 {
		return
	}
	_ = this.UpgradeV3(this.oldDir, false)
}

func (this *FileList) UpgradeV3(oldDir string, brokenOnError bool) error {
	// index.db
	var indexDBPath = oldDir + "/index.db"
	_, err := os.Stat(indexDBPath)
	if err != nil {
		return nil
	}
	remotelogs.Println("CACHE", "upgrading local database from '"+oldDir+"' ...")

	defer func() {
		_ = os.Remove(indexDBPath)
		remotelogs.Println("CACHE", "upgrading local database finished")
	}()

	db, err := dbs.OpenWriter("file:" + indexDBPath + "?cache=shared&mode=rwc&_journal_mode=WAL&_sync=" + dbs.SyncMode + "&_locking_mode=EXCLUSIVE")
	if err != nil {
		return err
	}

	defer func() {
		_ = db.Close()
	}()

	var isFinished = false
	var offset = 0
	var count = 10000

	for {
		if isFinished {
			break
		}

		err = func() error {
			defer func() {
				offset += count
			}()

			rows, err := db.Query(`SELECT "hash", "key", "headerSize", "bodySize", "metaSize", "expiredAt", "staleAt", "createdAt", "host", "serverId" FROM "cacheItems_v3" ORDER BY "id" ASC LIMIT ?, ?`, offset, count)
			if err != nil {
				return err
			}
			defer func() {
				_ = rows.Close()
			}()

			var hash = ""
			var key = ""
			var headerSize int64
			var bodySize int64
			var metaSize int64
			var expiredAt int64
			var staleAt int64
			var createdAt int64
			var host string
			var serverId int64

			isFinished = true

			for rows.Next() {
				isFinished = false

				err = rows.Scan(&hash, &key, &headerSize, &bodySize, &metaSize, &expiredAt, &staleAt, &createdAt, &host, &serverId)
				if err != nil {
					if brokenOnError {
						return err
					}
					return nil
				}

				err = this.Add(hash, &Item{
					Type:       ItemTypeFile,
					Key:        key,
					ExpiredAt:  expiredAt,
					StaleAt:    staleAt,
					HeaderSize: headerSize,
					BodySize:   bodySize,
					MetaSize:   metaSize,
					Host:       host,
					ServerId:   serverId,
				})
				if err != nil {
					if brokenOnError {
						return err
					}
				}
			}

			return nil
		}()
		if err != nil {
			return err
		}

		time.Sleep(1 * time.Second)
	}

	return nil
}

func (this *FileList) maxExpiresAtForMemoryCache(expiresAt int64) int64 {
	var maxTimestamp = fasttime.Now().Unix() + 3600
	if expiresAt > maxTimestamp {
		return maxTimestamp
	}
	return expiresAt
}
