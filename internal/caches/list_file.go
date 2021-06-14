// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package caches

import (
	"database/sql"
	"github.com/TeaOSLab/EdgeNode/internal/remotelogs"
	"github.com/TeaOSLab/EdgeNode/internal/utils"
	"github.com/iwind/TeaGo/lists"
	_ "github.com/mattn/go-sqlite3"
	"os"
	"strconv"
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

	existsByHashStmt *sql.Stmt // 根据hash检查是否存在
	insertStmt       *sql.Stmt // 写入数据
	selectByHashStmt *sql.Stmt // 使用hash查询数据
	deleteByHashStmt *sql.Stmt // 根据hash删除数据
	statStmt         *sql.Stmt // 统计
	purgeStmt        *sql.Stmt // 清理
	deleteAllStmt    *sql.Stmt // 删除所有数据

	oldTables      []string
	itemsTableName string

	isClosed bool
}

func NewFileList(dir string) ListInterface {
	return &FileList{dir: dir}
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

	this.itemsTableName = "cacheItems_v2"

	db, err := sql.Open("sqlite3", "file:"+this.dir+"/index.db?cache=shared&mode=rwc&_journal_mode=WAL")
	if err != nil {
		return err
	}
	db.SetMaxOpenConns(1)
	this.db = db

	// 清除旧表
	this.oldTables = []string{
		"cacheItems",
	}
	err = this.removeOldTables()
	if err != nil {
		remotelogs.Warn("CACHE", "clean old tables failed: "+err.Error())
	}

	// TODO 耗时过长，暂时不整理数据库
	/**_, err = db.Exec("VACUUM")
	if err != nil {
		return err
	}**/

	// 创建
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS "` + this.itemsTableName + `" (
  "id" integer NOT NULL PRIMARY KEY AUTOINCREMENT,
  "hash" varchar(32),
  "key" varchar(1024),
  "headerSize" integer DEFAULT 0,
  "bodySize" integer DEFAULT 0,
  "metaSize" integer DEFAULT 0,
  "expiredAt" integer DEFAULT 0,
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
		return err
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
	this.existsByHashStmt, err = this.db.Prepare(`SELECT "bodySize" FROM "` + this.itemsTableName + `" WHERE "hash"=? AND expiredAt>? LIMIT 1`)
	if err != nil {
		return err
	}

	this.insertStmt, err = this.db.Prepare(`INSERT INTO "` + this.itemsTableName + `" ("hash", "key", "headerSize", "bodySize", "metaSize", "expiredAt", "host", "serverId", "createdAt") VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`)
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

	this.statStmt, err = this.db.Prepare(`SELECT COUNT(*), IFNULL(SUM(headerSize+bodySize+metaSize), 0), IFNULL(SUM(headerSize+bodySize), 0) FROM "` + this.itemsTableName + `" WHERE expiredAt>?`)
	if err != nil {
		return err
	}

	this.purgeStmt, err = this.db.Prepare(`SELECT "hash" FROM "` + this.itemsTableName + `" WHERE expiredAt<=? LIMIT ?`)
	if err != nil {
		return err
	}

	this.deleteAllStmt, err = this.db.Prepare(`DELETE FROM "` + this.itemsTableName + `"`)
	if err != nil {
		return err
	}

	return nil
}

func (this *FileList) Reset() error {
	// 不错任何事情
	return nil
}

func (this *FileList) Add(hash string, item *Item) error {
	if this.isClosed {
		return nil
	}

	_, err := this.insertStmt.Exec(hash, item.Key, item.HeaderSize, item.BodySize, item.MetaSize, item.ExpiredAt, item.Host, item.ServerId, utils.UnixTime())
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
	if this.isClosed {
		return false, nil
	}

	rows, err := this.existsByHashStmt.Query(hash, time.Now().Unix())
	if err != nil {
		return false, err
	}
	defer func() {
		_ = rows.Close()
	}()
	if rows.Next() {
		return true, nil
	}
	return false, nil
}

// CleanPrefix 清理某个前缀的缓存数据
func (this *FileList) CleanPrefix(prefix string) error {
	if this.isClosed {
		return nil
	}

	if len(prefix) == 0 {
		return nil
	}

	var count = int64(10000)
	for {
		result, err := this.db.Exec(`UPDATE "`+this.itemsTableName+`" SET expiredAt=0 WHERE id IN (SELECT id FROM "`+this.itemsTableName+`" WHERE expiredAt>0 AND createdAt<=? AND INSTR("key", ?)==1 LIMIT `+strconv.FormatInt(count, 10)+`)`, utils.UnixTime(), prefix)
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
	if this.isClosed {
		return nil
	}

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

	atomic.AddInt64(&this.total, -1)

	if this.onRemove != nil {
		this.onRemove(item)
	}

	return nil
}

// Purge 清理过期的缓存
// count 每次遍历的最大数量，控制此数字可以保证每次清理的时候不用花太多时间
// callback 每次发现过期key的调用
func (this *FileList) Purge(count int, callback func(hash string) error) error {
	if this.isClosed {
		return nil
	}

	if count <= 0 {
		count = 1000
	}

	rows, err := this.purgeStmt.Query(time.Now().Unix(), count)
	if err != nil {
		return err
	}
	defer func() {
		_ = rows.Close()
	}()

	hashStrings := []string{}
	for rows.Next() {
		var hash string
		err = rows.Scan(&hash)
		if err != nil {
			return err
		}
		hashStrings = append(hashStrings, hash)
	}

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
	if this.isClosed {
		return nil
	}

	_, err := this.deleteAllStmt.Exec()
	if err != nil {
		return err
	}
	atomic.StoreInt64(&this.total, 0)
	return nil
}

func (this *FileList) Stat(check func(hash string) bool) (*Stat, error) {
	if this.isClosed {
		return &Stat{}, nil
	}

	// 这里不设置过期时间、不使用 check 函数，目的是让查询更快速一些
	row := this.statStmt.QueryRow(time.Now().Unix())
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

	if this.db != nil {
		_ = this.existsByHashStmt.Close()
		_ = this.insertStmt.Close()
		_ = this.selectByHashStmt.Close()
		_ = this.deleteByHashStmt.Close()
		_ = this.statStmt.Close()
		_ = this.purgeStmt.Close()
		_ = this.deleteAllStmt.Close()

		return this.db.Close()
	}
	return nil
}

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
			go func() {
				remotelogs.Println("CACHE", "remove old table '"+name+"' ...")
				_, _ = this.db.Exec(`DROP TABLE "` + name + `"`)
				remotelogs.Println("CACHE", "remove old table '"+name+"' done")
			}()
		}

	}
	return nil
}
