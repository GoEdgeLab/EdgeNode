// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package caches

import (
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
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
}

func NewFileList(dir string) ListInterface {
	return &FileList{dir: dir}
}

func (this *FileList) Init() error {
	db, err := sql.Open("sqlite3", "file:"+this.dir+"/index.db?cache=shared&mode=rwc")
	if err != nil {
		return err
	}
	db.SetMaxOpenConns(1)

	_, err = db.Exec("VACUUM")
	if err != nil {
		return err
	}

	// 创建
	// TODO accessesAt 用来存储访问时间，将来可以根据此访问时间删除不常访问的内容
	//   且访问时间只需要每隔一个小时存储一个整数值即可，因为不需要那么精确
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS "cacheItems" (
  "hash" varchar(32),
  "key" varchar(1024),
  "headerSize" integer DEFAULT 0,
  "bodySize" integer DEFAULT 0,
  "metaSize" integer DEFAULT 0,
  "expiredAt" integer DEFAULT 0,
  "accessedAt" integer DEFAULT 0
);

CREATE UNIQUE INDEX IF NOT EXISTS "hash"
ON "cacheItems" (
  "hash"
);
CREATE INDEX IF NOT EXISTS "expiredAt"
ON  "cacheItems" (
  "expiredAt"
);
CREATE INDEX IF NOT EXISTS "accessedAt"
ON  "cacheItems" (
  "accessedAt"
);
`)
	if err != nil {
		return err
	}

	this.db = db

	// 读取总数量
	row := this.db.QueryRow("SELECT COUNT(*) FROM cacheItems")
	if row.Err() != nil {
		return row.Err()
	}
	var total int64
	err = row.Scan(&total)
	if err != nil {
		return err
	}
	this.total = total

	return nil
}

func (this *FileList) Reset() error {
	// 不错任何事情
	return nil
}

func (this *FileList) Add(hash string, item *Item) error {
	_, err := this.db.Exec(`INSERT INTO cacheItems ("hash", "key", "headerSize", "bodySize", "metaSize", "expiredAt") VALUES (?, ?, ?, ?, ?, ?)`, hash, item.Key, item.HeaderSize, item.BodySize, item.MetaSize, item.ExpiredAt)
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
	row := this.db.QueryRow(`SELECT "bodySize" FROM cacheItems WHERE "hash"=? LIMIT 1`, hash)
	if row == nil {
		return false, nil
	}
	if row.Err() != nil {
		return false, row.Err()
	}
	var bodySize int
	err := row.Scan(&bodySize)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// FindKeysWithPrefix 根据前缀进行查找
func (this *FileList) FindKeysWithPrefix(prefix string) (keys []string, err error) {
	if len(prefix) == 0 {
		return
	}

	// TODO 需要优化上千万结果的情况

	rows, err := this.db.Query(`SELECT "key" FROM cacheItems WHERE INSTR("key", ?)==1 LIMIT 100000`, prefix)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rows.Close()
	}()

	for rows.Next() {
		var key string
		err = rows.Scan(&key)
		if err != nil {
			return nil, err
		}
		keys = append(keys, key)
	}

	return
}

func (this *FileList) Remove(hash string) error {
	row := this.db.QueryRow(`SELECT "key", "headerSize", "bodySize", "metaSize", "expiredAt" FROM cacheItems WHERE "hash"=? LIMIT 1`, hash)
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

	_, err = this.db.Exec(`DELETE FROM cacheItems WHERE "hash"=?`, hash)
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
	if count <= 0 {
		count = 1000
	}

	rows, err := this.db.Query(`SELECT "hash" FROM cacheItems WHERE expiredAt<=? LIMIT ?`, time.Now().Unix(), count)
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
	_, err := this.db.Exec("DELETE FROM cacheItems")
	if err != nil {
		return err
	}
	atomic.StoreInt64(&this.total, 0)
	return nil
}

func (this *FileList) Stat(check func(hash string) bool) (*Stat, error) {
	// 这里不设置过期时间、不使用 check 函数，目的是让查询更快速一些
	row := this.db.QueryRow("SELECT COUNT(*), IFNULL(SUM(headerSize+bodySize+metaSize), 0), IFNULL(SUM(headerSize+bodySize), 0) FROM cacheItems")
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
