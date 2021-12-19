// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package caches

import (
	"database/sql"
	"errors"
	"github.com/TeaOSLab/EdgeNode/internal/goman"
	"github.com/TeaOSLab/EdgeNode/internal/remotelogs"
	"github.com/TeaOSLab/EdgeNode/internal/ttlcache"
	"github.com/TeaOSLab/EdgeNode/internal/utils"
	"github.com/iwind/TeaGo/lists"
	"github.com/iwind/TeaGo/types"
	timeutil "github.com/iwind/TeaGo/utils/time"
	_ "github.com/mattn/go-sqlite3"
	"os"
	"sync/atomic"
	"time"
)

// FileList 文件缓存列表管理
type FileList struct {
	dir   string
	db    *sql.DB
	total int64

	onAdd    func(item *Item)
	onRemove func(item *Item)

	// cacheItems
	existsByHashStmt *sql.Stmt // 根据hash检查是否存在
	insertStmt       *sql.Stmt // 写入数据
	selectByHashStmt *sql.Stmt // 使用hash查询数据
	deleteByHashStmt *sql.Stmt // 根据hash删除数据
	statStmt         *sql.Stmt // 统计
	purgeStmt        *sql.Stmt // 清理
	deleteAllStmt    *sql.Stmt // 删除所有数据

	// hits
	insertHitStmt       *sql.Stmt // 写入数据
	increaseHitStmt     *sql.Stmt // 增加点击量
	deleteHitByHashStmt *sql.Stmt // 根据hash删除数据
	lfuHitsStmt         *sql.Stmt // 读取老的数据

	oldTables      []string
	itemsTableName string
	hitsTableName  string

	isClosed bool
	isReady  bool

	memoryCache *ttlcache.Cache
}

func NewFileList(dir string) ListInterface {
	return &FileList{
		dir:         dir,
		memoryCache: ttlcache.NewCache(),
	}
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

	this.itemsTableName = "cacheItems_v3"
	this.hitsTableName = "hits"

	var dir = this.dir
	if dir == "/" {
		// 防止sqlite提示authority错误
		dir = ""
	}
	var dbPath = dir + "/index.db"
	remotelogs.Println("CACHE", "loading database '"+dbPath+"'")
	db, err := sql.Open("sqlite3", "file:"+dbPath+"?cache=shared&mode=rwc&_journal_mode=WAL")
	if err != nil {
		return errors.New("open database failed: " + err.Error())
	}

	db.SetMaxOpenConns(1)

	this.db = db

	// TODO 耗时过长，暂时不整理数据库
	/**_, err = db.Exec("VACUUM")
	if err != nil {
		return err
	}**/

	// 创建
	err = this.initTables(db, 1)
	if err != nil {
		return errors.New("init tables failed: " + err.Error())
	}

	// 清除旧表
	// 这个一定要在initTables()之后，因为老的数据需要转移
	this.oldTables = []string{
		"cacheItems",
		"cacheItems_v2",
	}
	err = this.removeOldTables()
	if err != nil {
		remotelogs.Warn("CACHE", "clean old tables failed: "+err.Error())
	}

	// 读取总数量
	row := this.db.QueryRow(`SELECT COUNT(*) FROM "` + this.itemsTableName + `"`)
	if row.Err() != nil {
		return row.Err()
	}
	var total int64
	err = row.Scan(&total)
	if err != nil {
		return err
	}
	this.total = total

	// 常用语句
	this.existsByHashStmt, err = this.db.Prepare(`SELECT "expiredAt" FROM "` + this.itemsTableName + `" WHERE "hash"=? AND expiredAt>? LIMIT 1`)
	if err != nil {
		return err
	}

	this.insertStmt, err = this.db.Prepare(`INSERT INTO "` + this.itemsTableName + `" ("hash", "key", "headerSize", "bodySize", "metaSize", "expiredAt", "staleAt", "host", "serverId", "createdAt") VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}

	this.selectByHashStmt, err = this.db.Prepare(`SELECT "key", "headerSize", "bodySize", "metaSize", "expiredAt" FROM "` + this.itemsTableName + `" WHERE "hash"=? LIMIT 1`)
	if err != nil {
		return err
	}

	this.deleteByHashStmt, err = this.db.Prepare(`DELETE FROM "` + this.itemsTableName + `" WHERE "hash"=?`)
	if err != nil {
		return err
	}

	this.statStmt, err = this.db.Prepare(`SELECT COUNT(*), IFNULL(SUM(headerSize+bodySize+metaSize), 0), IFNULL(SUM(headerSize+bodySize), 0) FROM "` + this.itemsTableName + `"`)
	if err != nil {
		return err
	}

	this.purgeStmt, err = this.db.Prepare(`SELECT "hash" FROM "` + this.itemsTableName + `" WHERE staleAt<=? LIMIT ?`)
	if err != nil {
		return err
	}

	this.deleteAllStmt, err = this.db.Prepare(`DELETE FROM "` + this.itemsTableName + `"`)
	if err != nil {
		return err
	}

	this.insertHitStmt, err = this.db.Prepare(`INSERT INTO "` + this.hitsTableName + `" ("hash", "week2Hits", "week") VALUES (?, 1, ?)`)

	this.increaseHitStmt, err = this.db.Prepare(`INSERT INTO "` + this.hitsTableName + `" ("hash", "week2Hits", "week") VALUES (?, 1, ?) ON CONFLICT("hash") DO UPDATE SET "week1Hits"=IIF("week"=?, "week1Hits", "week2Hits"), "week2Hits"=IIF("week"=?, "week2Hits"+1, 1), "week"=?`)
	if err != nil {
		return err
	}

	this.deleteHitByHashStmt, err = this.db.Prepare(`DELETE FROM "` + this.hitsTableName + `" WHERE "hash"=?`)
	if err != nil {
		return err
	}

	this.lfuHitsStmt, err = this.db.Prepare(`SELECT "hash" FROM "` + this.hitsTableName + `" ORDER BY "week" ASC, "week1Hits"+"week2Hits" ASC LIMIT ?`)
	if err != nil {
		return err
	}

	this.isReady = true

	return nil
}

func (this *FileList) Reset() error {
	// 不做任何事情
	return nil
}

func (this *FileList) Add(hash string, item *Item) error {
	if !this.isReady {
		return nil
	}

	if item.StaleAt == 0 {
		item.StaleAt = item.ExpiredAt
	}

	_, err := this.insertStmt.Exec(hash, item.Key, item.HeaderSize, item.BodySize, item.MetaSize, item.ExpiredAt, item.StaleAt, item.Host, item.ServerId, utils.UnixTime())
	if err != nil {
		return err
	}

	_, err = this.insertHitStmt.Exec(hash, timeutil.Format("YW"))
	if err != nil {
		return err
	}

	atomic.AddInt64(&this.total, 1)

	if this.onAdd != nil {
		this.onAdd(item)
	}
	return nil
}

func (this *FileList) Exist(hash string) (bool, error) {
	if !this.isReady {
		return false, nil
	}

	item := this.memoryCache.Read(hash)
	if item != nil {
		return true, nil
	}

	rows, err := this.existsByHashStmt.Query(hash, time.Now().Unix())
	if err != nil {
		return false, err
	}
	defer func() {
		_ = rows.Close()
	}()
	if rows.Next() {
		var expiredAt int64
		err = rows.Scan(&expiredAt)
		if err != nil {
			return false, nil
		}
		this.memoryCache.Write(hash, 1, expiredAt)
		return true, nil
	}
	return false, nil
}

// CleanPrefix 清理某个前缀的缓存数据
func (this *FileList) CleanPrefix(prefix string) error {
	if !this.isReady {
		return nil
	}

	if len(prefix) == 0 {
		return nil
	}

	defer func() {
		this.memoryCache.Clean()
	}()

	var count = int64(10000)
	var staleLife = 600 // TODO 需要可以设置
	for {
		result, err := this.db.Exec(`UPDATE "`+this.itemsTableName+`" SET expiredAt=0,staleAt=? WHERE id IN (SELECT id FROM "`+this.itemsTableName+`" WHERE expiredAt>0 AND createdAt<=? AND INSTR("key", ?)=1 LIMIT `+types.String(count)+`)`, utils.UnixTime()+int64(staleLife), utils.UnixTime(), prefix)
		if err != nil {
			return err
		}
		affectedRows, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if affectedRows < count {
			return nil
		}
	}
}

func (this *FileList) Remove(hash string) error {
	if !this.isReady {
		return nil
	}

	// 从缓存中删除
	this.memoryCache.Delete(hash)

	row := this.selectByHashStmt.QueryRow(hash)
	if row.Err() != nil {
		return row.Err()
	}

	var item = &Item{Type: ItemTypeFile}
	err := row.Scan(&item.Key, &item.HeaderSize, &item.BodySize, &item.MetaSize, &item.ExpiredAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		return err
	}

	_, err = this.deleteByHashStmt.Exec(hash)
	if err != nil {
		return err
	}

	_, err = this.deleteHitByHashStmt.Exec(hash)
	if err != nil {
		return err
	}

	atomic.AddInt64(&this.total, -1)

	if this.onRemove != nil {
		this.onRemove(item)
	}

	return nil
}

// Purge 清理过期的缓存
// count 每次遍历的最大数量，控制此数字可以保证每次清理的时候不用花太多时间
// callback 每次发现过期key的调用
func (this *FileList) Purge(count int, callback func(hash string) error) (int, error) {
	if !this.isReady {
		return 0, nil
	}

	if count <= 0 {
		count = 1000
	}

	rows, err := this.purgeStmt.Query(time.Now().Unix(), count)
	if err != nil {
		return 0, err
	}

	hashStrings := []string{}
	var countFound = 0
	for rows.Next() {
		var hash string
		err = rows.Scan(&hash)
		if err != nil {
			_ = rows.Close()
			return 0, err
		}
		hashStrings = append(hashStrings, hash)
		countFound++
	}
	_ = rows.Close() // 不能使用defer，防止读写冲突

	// 不在 rows.Next() 循环中操作是为了避免死锁
	for _, hash := range hashStrings {
		err = this.Remove(hash)
		if err != nil {
			return 0, err
		}

		err = callback(hash)
		if err != nil {
			return 0, err
		}
	}

	return countFound, nil
}

func (this *FileList) PurgeLFU(count int, callback func(hash string) error) error {
	if !this.isReady {
		return nil
	}

	if count <= 0 {
		return nil
	}

	rows, err := this.lfuHitsStmt.Query(count)
	if err != nil {
		return err
	}

	hashStrings := []string{}
	var countFound = 0
	for rows.Next() {
		var hash string
		err = rows.Scan(&hash)
		if err != nil {
			_ = rows.Close()
			return err
		}
		hashStrings = append(hashStrings, hash)
		countFound++
	}
	_ = rows.Close() // 不能使用defer，防止读写冲突

	// 不在 rows.Next() 循环中操作是为了避免死锁
	for _, hash := range hashStrings {
		err = this.Remove(hash)
		if err != nil {
			return err
		}

		err = callback(hash)
		if err != nil {
			return err
		}
	}
	return nil
}

func (this *FileList) CleanAll() error {
	if !this.isReady {
		return nil
	}

	this.memoryCache.Clean()

	_, err := this.deleteAllStmt.Exec()
	if err != nil {
		return err
	}
	atomic.StoreInt64(&this.total, 0)
	return nil
}

func (this *FileList) Stat(check func(hash string) bool) (*Stat, error) {
	if !this.isReady {
		return &Stat{}, nil
	}

	// 这里不设置过期时间、不使用 check 函数，目的是让查询更快速一些
	row := this.statStmt.QueryRow()
	if row.Err() != nil {
		return nil, row.Err()
	}
	stat := &Stat{}
	err := row.Scan(&stat.Count, &stat.Size, &stat.ValueSize)
	if err != nil {
		return nil, err
	}

	return stat, nil
}

// Count 总数量
// 常用的方法，所以避免直接查询数据库
func (this *FileList) Count() (int64, error) {
	return atomic.LoadInt64(&this.total), nil
}

// IncreaseHit 增加点击量
func (this *FileList) IncreaseHit(hash string) error {
	var week = timeutil.Format("YW")
	_, err := this.increaseHitStmt.Exec(hash, week, week, week, week)
	return err
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
	this.isClosed = true
	this.isReady = false

	this.memoryCache.Destroy()

	if this.db != nil {
		_ = this.existsByHashStmt.Close()
		_ = this.insertStmt.Close()
		_ = this.selectByHashStmt.Close()
		_ = this.deleteByHashStmt.Close()
		_ = this.statStmt.Close()
		_ = this.purgeStmt.Close()
		_ = this.deleteAllStmt.Close()

		_ = this.insertHitStmt.Close()
		_ = this.increaseHitStmt.Close()
		_ = this.deleteHitByHashStmt.Close()
		_ = this.lfuHitsStmt.Close()

		return this.db.Close()
	}
	return nil
}

// 初始化
func (this *FileList) initTables(db *sql.DB, times int) error {
	// 检查是否存在
	_, err := db.Exec(`SELECT id FROM "` + this.itemsTableName + `" LIMIT 1`)
	var notFound = false
	if err != nil {
		notFound = true
	}

	{
		// expiredAt - 过期时间，用来判断有无过期
		// staleAt - 陈旧最大时间，用来清理缓存
		_, err := db.Exec(`CREATE TABLE IF NOT EXISTS "` + this.itemsTableName + `" (
  "id" integer NOT NULL PRIMARY KEY AUTOINCREMENT,
  "hash" varchar(32),
  "key" varchar(1024),
  "headerSize" integer DEFAULT 0,
  "bodySize" integer DEFAULT 0,
  "metaSize" integer DEFAULT 0,
  "expiredAt" integer DEFAULT 0,
  "staleAt" integer DEFAULT 0,
  "createdAt" integer DEFAULT 0,
  "host" varchar(128),
  "serverId" integer
);

CREATE INDEX IF NOT EXISTS "createdAt"
ON "` + this.itemsTableName + `" (
  "createdAt" ASC
);

CREATE INDEX IF NOT EXISTS "expiredAt"
ON "` + this.itemsTableName + `" (
  "expiredAt" ASC
);

CREATE INDEX IF NOT EXISTS "staleAt"
ON "` + this.itemsTableName + `" (
  "staleAt" ASC
);

CREATE UNIQUE INDEX IF NOT EXISTS "hash"
ON "` + this.itemsTableName + `" (
  "hash" ASC
);

CREATE INDEX IF NOT EXISTS "serverId"
ON "` + this.itemsTableName + `" (
  "serverId" ASC
);
`)

		if err != nil {
			// 尝试删除重建
			if times < 3 {
				_, dropErr := db.Exec(`DROP TABLE "` + this.itemsTableName + `"`)
				if dropErr == nil {
					return this.initTables(db, times+1)
				}
				return err
			}

			return err
		}
	}

	// 如果数据为空，从老数据中加载数据
	if notFound {
		// v2 => v3
		remotelogs.Println("CACHE", "transferring old data from v2 to v3 ...")
		result, err := db.Exec(`INSERT INTO "` + this.itemsTableName + `" ("id", "hash", "key", "headerSize", "bodySize", "metaSize", "expiredAt", "createdAt", "host", "serverId", "staleAt") SELECT "id", "hash", "key", "headerSize", "bodySize", "metaSize", "expiredAt", "createdAt", "host", "serverId", "expiredAt"+600 FROM cacheItems_v2`)
		if err != nil {
			remotelogs.Println("CACHE", "transfer old data from v2 to v3 failed: "+err.Error())
		} else {
			count, _ := result.RowsAffected()
			remotelogs.Println("CACHE", "transfer old data from v2 to v3 finished, "+types.String(count)+" rows transferred")
		}
	}

	{
		_, err := db.Exec(`CREATE TABLE IF NOT EXISTS "` + this.hitsTableName + `" (
  "id" integer NOT NULL PRIMARY KEY AUTOINCREMENT,
  "hash" varchar(32),
  "week1Hits" integer DEFAULT 0,
  "week2Hits" integer DEFAULT 0,
  "week" varchar(6)
);

CREATE UNIQUE INDEX IF NOT EXISTS "hits_hash"
ON "` + this.hitsTableName + `" (
  "hash" ASC
);
`)
		if err != nil {
			// 尝试删除重建
			if times < 3 {
				_, dropErr := db.Exec(`DROP TABLE "` + this.hitsTableName + `"`)
				if dropErr == nil {
					return this.initTables(db, times+1)
				}
				return err
			}

			return err
		}
	}

	return nil
}

// 删除过期不用的表格
func (this *FileList) removeOldTables() error {
	rows, err := this.db.Query(`SELECT "name" FROM sqlite_master WHERE "type"='table'`)
	if err != nil {
		return err
	}
	defer func() {
		_ = rows.Close()
	}()
	for rows.Next() {
		var name string
		err = rows.Scan(&name)
		if err != nil {
			return err
		}
		if lists.ContainsString(this.oldTables, name) {
			// 异步执行
			goman.New(func() {
				remotelogs.Println("CACHE", "remove old table '"+name+"' ...")
				_, _ = this.db.Exec(`DROP TABLE "` + name + `"`)
				remotelogs.Println("CACHE", "remove old table '"+name+"' done")
			})
		}

	}
	return nil
}
