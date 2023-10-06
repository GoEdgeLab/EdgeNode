// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package caches

import (
	"errors"
	"fmt"
	teaconst "github.com/TeaOSLab/EdgeNode/internal/const"
	"github.com/TeaOSLab/EdgeNode/internal/remotelogs"
	"github.com/TeaOSLab/EdgeNode/internal/utils"
	"github.com/TeaOSLab/EdgeNode/internal/utils/dbs"
	"github.com/TeaOSLab/EdgeNode/internal/utils/fasttime"
	"github.com/iwind/TeaGo/logs"
	"github.com/iwind/TeaGo/types"
	timeutil "github.com/iwind/TeaGo/utils/time"
	"net"
	"net/url"
	"os"
	"runtime"
	"strings"
	"time"
)

type FileListDB struct {
	dbPath string

	readDB  *dbs.DB
	writeDB *dbs.DB

	hashMap *FileListHashMap

	itemsTableName string

	isClosed        bool // 是否已关闭
	isReady         bool // 是否已完成初始化
	hashMapIsLoaded bool // Hash是否已加载

	// cacheItems
	existsByHashStmt *dbs.Stmt // 根据hash检查是否存在

	insertStmt *dbs.Stmt // 写入数据
	insertSQL  string

	selectByHashStmt *dbs.Stmt // 使用hash查询数据

	selectHashListStmt *dbs.Stmt

	deleteByHashStmt *dbs.Stmt // 根据hash删除数据
	deleteByHashSQL  string

	statStmt             *dbs.Stmt // 统计
	purgeStmt            *dbs.Stmt // 清理
	deleteAllStmt        *dbs.Stmt // 删除所有数据
	listOlderItemsStmt   *dbs.Stmt // 读取较早存储的缓存
	updateAccessWeekStmt *dbs.Stmt // 修改访问日期
}

func NewFileListDB() *FileListDB {
	return &FileListDB{
		hashMap: NewFileListHashMap(),
	}
}

func (this *FileListDB) Open(dbPath string) error {
	this.dbPath = dbPath

	// 动态调整Cache值
	var cacheSize = 32000
	var memoryGB = utils.SystemMemoryGB()
	if memoryGB >= 8 {
		cacheSize += 32000 * memoryGB / 8
	}

	// write db
	// 这里不能加 EXCLUSIVE 锁，不然异步事务可能会失败
	writeDB, err := dbs.OpenWriter("file:" + dbPath + "?cache=private&mode=rwc&_journal_mode=WAL&_sync=" + dbs.SyncMode + "&_cache_size=" + types.String(cacheSize) + "&_secure_delete=FAST")
	if err != nil {
		return fmt.Errorf("open write database failed: %w", err)
	}

	writeDB.SetMaxOpenConns(1)

	this.writeDB = writeDB

	// TODO 耗时过长，暂时不整理数据库
	// TODO 需要根据行数来判断是否VACUUM
	// TODO 注意VACUUM反而可能让数据库文件变大
	/**_, err = db.Exec("VACUUM")
	if err != nil {
		return err
	}**/

	// 检查是否损坏
	// TODO 暂时屏蔽，因为用时过长

	var recoverEnv, _ = os.LookupEnv("EdgeRecover")
	if len(recoverEnv) > 0 && this.shouldRecover() {
		for _, indexName := range []string{"staleAt", "hash"} {
			_, _ = this.writeDB.Exec(`REINDEX "` + indexName + `"`)
		}
	}

	if teaconst.EnableDBStat {
		this.writeDB.EnableStat(true)
	}

	// read db
	readDB, err := dbs.OpenReader("file:" + dbPath + "?cache=private&mode=ro&_journal_mode=WAL&_sync=OFF&_cache_size=" + types.String(cacheSize))
	if err != nil {
		return fmt.Errorf("open read database failed: %w", err)
	}

	readDB.SetMaxOpenConns(runtime.NumCPU())

	this.readDB = readDB

	if teaconst.EnableDBStat {
		this.readDB.EnableStat(true)
	}

	return nil
}

func (this *FileListDB) Init() error {
	this.itemsTableName = "cacheItems"

	// 创建
	var err = this.initTables(1)
	if err != nil {
		return fmt.Errorf("init tables failed: %w", err)
	}

	// 常用语句
	this.existsByHashStmt, err = this.readDB.Prepare(`SELECT "expiredAt" FROM "` + this.itemsTableName + `" INDEXED BY "hash" WHERE "hash"=? AND expiredAt>? LIMIT 1`)
	if err != nil {
		return err
	}

	this.insertSQL = `INSERT INTO "` + this.itemsTableName + `" ("hash", "key", "headerSize", "bodySize", "metaSize", "expiredAt", "staleAt", "host", "serverId", "createdAt", "accessWeek") VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	this.insertStmt, err = this.writeDB.Prepare(this.insertSQL)
	if err != nil {
		return err
	}

	this.selectByHashStmt, err = this.readDB.Prepare(`SELECT "key", "headerSize", "bodySize", "metaSize", "expiredAt" FROM "` + this.itemsTableName + `" WHERE "hash"=? LIMIT 1`)
	if err != nil {
		return err
	}

	this.selectHashListStmt, err = this.readDB.Prepare(`SELECT "id", "hash" FROM "` + this.itemsTableName + `" WHERE id>? ORDER BY id ASC LIMIT 2000`)
	if err != nil {
		return err
	}

	this.deleteByHashSQL = `DELETE FROM "` + this.itemsTableName + `" WHERE "hash"=?`
	this.deleteByHashStmt, err = this.writeDB.Prepare(this.deleteByHashSQL)
	if err != nil {
		return err
	}

	this.statStmt, err = this.readDB.Prepare(`SELECT COUNT(*), IFNULL(SUM(headerSize+bodySize+metaSize), 0), IFNULL(SUM(headerSize+bodySize), 0) FROM "` + this.itemsTableName + `"`)
	if err != nil {
		return err
	}

	this.purgeStmt, err = this.readDB.Prepare(`SELECT "hash" FROM "` + this.itemsTableName + `" WHERE staleAt<=? LIMIT ?`)
	if err != nil {
		return err
	}

	this.deleteAllStmt, err = this.writeDB.Prepare(`DELETE FROM "` + this.itemsTableName + `"`)
	if err != nil {
		return err
	}

	this.updateAccessWeekStmt, err = this.writeDB.Prepare(`UPDATE "` + this.itemsTableName + `" SET "accessWeek"=? WHERE "hash"=?`)
	if err != nil {
		return err
	}

	this.listOlderItemsStmt, err = this.readDB.Prepare(`SELECT "hash" FROM "` + this.itemsTableName + `" ORDER BY "accessWeek" ASC, "id" ASC LIMIT ?`)
	if err != nil {
		return err
	}

	this.isReady = true

	// 加载HashMap
	go this.loadHashMap()

	return nil
}

func (this *FileListDB) IsReady() bool {
	return this.isReady
}

func (this *FileListDB) Total() (int64, error) {
	// 读取总数量
	var row = this.readDB.QueryRow(`SELECT COUNT(*) FROM "` + this.itemsTableName + `"`)
	if row.Err() != nil {
		return 0, row.Err()
	}
	var total int64
	err := row.Scan(&total)
	return total, err
}

func (this *FileListDB) AddSync(hash string, item *Item) error {
	this.hashMap.Add(hash)

	if item.StaleAt == 0 {
		item.StaleAt = item.ExpiredAt
	}

	_, err := this.insertStmt.Exec(hash, item.Key, item.HeaderSize, item.BodySize, item.MetaSize, item.ExpiredAt, item.StaleAt, item.Host, item.ServerId, fasttime.Now().Unix(), timeutil.Format("YW"))
	if err != nil {
		return this.WrapError(err)
	}

	return nil
}

func (this *FileListDB) DeleteSync(hash string) error {
	this.hashMap.Delete(hash)

	_, err := this.deleteByHashStmt.Exec(hash)
	if err != nil {
		return err
	}
	return nil
}

func (this *FileListDB) ListExpiredItems(count int) (hashList []string, err error) {
	if !this.isReady {
		return nil, nil
	}

	if count <= 0 {
		count = 100
	}

	rows, err := this.purgeStmt.Query(time.Now().Unix(), count)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rows.Close()
	}()

	for rows.Next() {
		var hash string
		err = rows.Scan(&hash)
		if err != nil {
			return nil, err
		}
		hashList = append(hashList, hash)
	}
	return hashList, nil
}

func (this *FileListDB) ListLFUItems(count int) (hashList []string, err error) {
	if !this.isReady {
		return nil, nil
	}

	if count <= 0 {
		count = 100
	}

	// 先找过期的
	hashList, err = this.ListExpiredItems(count)
	if err != nil {
		return
	}
	var l = len(hashList)

	// 从旧缓存中补充
	if l < count {
		oldHashList, err := this.listOlderItems(count - l)
		if err != nil {
			return nil, err
		}
		hashList = append(hashList, oldHashList...)
	}

	return hashList, nil
}

func (this *FileListDB) ListHashes(lastId int64) (hashList []string, maxId int64, err error) {
	rows, err := this.selectHashListStmt.Query(lastId)
	if err != nil {
		return nil, 0, err
	}
	var id int64
	var hash string
	for rows.Next() {
		err = rows.Scan(&id, &hash)
		if err != nil {
			_ = rows.Close()
			return
		}
		maxId = id
		hashList = append(hashList, hash)
	}

	_ = rows.Close()
	return
}

func (this *FileListDB) IncreaseHitAsync(hash string) error {
	_, err := this.updateAccessWeekStmt.Exec(timeutil.Format("YW"), hash)
	return err
}

func (this *FileListDB) CleanPrefix(prefix string) error {
	if !this.isReady {
		return nil
	}
	var count = int64(10000)
	var unixTime = fasttime.Now().Unix() // 只删除当前的，不删除新的
	for {
		result, err := this.writeDB.Exec(`UPDATE "`+this.itemsTableName+`" SET expiredAt=0,staleAt=? WHERE id IN (SELECT id FROM "`+this.itemsTableName+`" WHERE expiredAt>0 AND createdAt<=? AND INSTR("key", ?)=1 LIMIT `+types.String(count)+`)`, unixTime+DefaultStaleCacheSeconds, unixTime, prefix)
		if err != nil {
			return this.WrapError(err)
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

func (this *FileListDB) CleanMatchKey(key string) error {
	if !this.isReady {
		return nil
	}

	// 忽略 @GOEDGE_
	if strings.Contains(key, SuffixAll) {
		return nil
	}

	u, err := url.Parse(key)
	if err != nil {
		return nil
	}

	var host = u.Host
	hostPart, _, err := net.SplitHostPort(host)
	if err == nil && len(hostPart) > 0 {
		host = hostPart
	}
	if len(host) == 0 {
		return nil
	}

	// 转义
	var queryKey = strings.ReplaceAll(key, "%", "\\%")
	queryKey = strings.ReplaceAll(queryKey, "_", "\\_")
	queryKey = strings.Replace(queryKey, "*", "%", 1)

	// TODO 检查大批量数据下的操作性能
	var unixTime = fasttime.Now().Unix() // 只删除当前的，不删除新的

	_, err = this.writeDB.Exec(`UPDATE "`+this.itemsTableName+`" SET "expiredAt"=0, "staleAt"=? WHERE "host" GLOB ? AND "host" NOT GLOB ? AND "key" LIKE ? ESCAPE '\'`, unixTime+DefaultStaleCacheSeconds, host, "*."+host, queryKey)
	if err != nil {
		return err
	}

	_, err = this.writeDB.Exec(`UPDATE "`+this.itemsTableName+`" SET "expiredAt"=0, "staleAt"=? WHERE "host" GLOB ? AND "host" NOT GLOB ? AND "key" LIKE ? ESCAPE '\'`, unixTime+DefaultStaleCacheSeconds, host, "*."+host, queryKey+SuffixAll+"%")
	if err != nil {
		return err
	}

	return nil
}

func (this *FileListDB) CleanMatchPrefix(prefix string) error {
	if !this.isReady {
		return nil
	}

	u, err := url.Parse(prefix)
	if err != nil {
		return nil
	}

	var host = u.Host
	hostPart, _, err := net.SplitHostPort(host)
	if err == nil && len(hostPart) > 0 {
		host = hostPart
	}
	if len(host) == 0 {
		return nil
	}

	// 转义
	var queryPrefix = strings.ReplaceAll(prefix, "%", "\\%")
	queryPrefix = strings.ReplaceAll(queryPrefix, "_", "\\_")
	queryPrefix = strings.Replace(queryPrefix, "*", "%", 1)
	queryPrefix += "%"

	// TODO 检查大批量数据下的操作性能
	var unixTime = fasttime.Now().Unix() // 只删除当前的，不删除新的

	_, err = this.writeDB.Exec(`UPDATE "`+this.itemsTableName+`" SET "expiredAt"=0, "staleAt"=? WHERE "host" GLOB ? AND "host" NOT GLOB ? AND "key" LIKE ? ESCAPE '\'`, unixTime+DefaultStaleCacheSeconds, host, "*."+host, queryPrefix)
	return err
}

func (this *FileListDB) CleanAll() error {
	if !this.isReady {
		return nil
	}

	_, err := this.deleteAllStmt.Exec()
	if err != nil {
		return this.WrapError(err)
	}

	this.hashMap.Clean()

	return nil
}

func (this *FileListDB) Close() error {
	if this.isClosed {
		return nil
	}

	this.isClosed = true
	this.isReady = false

	if this.existsByHashStmt != nil {
		_ = this.existsByHashStmt.Close()
	}
	if this.insertStmt != nil {
		_ = this.insertStmt.Close()
	}
	if this.selectByHashStmt != nil {
		_ = this.selectByHashStmt.Close()
	}
	if this.selectHashListStmt != nil {
		_ = this.selectHashListStmt.Close()
	}
	if this.deleteByHashStmt != nil {
		_ = this.deleteByHashStmt.Close()
	}
	if this.statStmt != nil {
		_ = this.statStmt.Close()
	}
	if this.purgeStmt != nil {
		_ = this.purgeStmt.Close()
	}
	if this.deleteAllStmt != nil {
		_ = this.deleteAllStmt.Close()
	}
	if this.updateAccessWeekStmt != nil {
		_ = this.updateAccessWeekStmt.Close()
	}
	if this.listOlderItemsStmt != nil {
		_ = this.listOlderItemsStmt.Close()
	}

	var errStrings []string

	if this.readDB != nil {
		err := this.readDB.Close()
		if err != nil {
			errStrings = append(errStrings, err.Error())
		}
	}

	if this.writeDB != nil {
		err := this.writeDB.Close()
		if err != nil {
			errStrings = append(errStrings, err.Error())
		}
	}

	if len(errStrings) == 0 {
		return nil
	}
	return errors.New("close database failed: " + strings.Join(errStrings, ", "))
}

func (this *FileListDB) WrapError(err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%w (file: %s)", err, this.dbPath)
}

func (this *FileListDB) HashMapIsLoaded() bool {
	return this.hashMapIsLoaded
}

// 初始化
func (this *FileListDB) initTables(times int) error {
	{
		// expiredAt - 过期时间，用来判断有无过期
		// staleAt - 过时缓存最大时间，用来清理缓存
		// 不对 hash 增加 unique 参数，是尽可能避免产生 malformed 错误
		_, err := this.writeDB.Exec(`CREATE TABLE IF NOT EXISTS "` + this.itemsTableName + `" (
  "id" integer NOT NULL PRIMARY KEY AUTOINCREMENT,
  "hash" varchar(32),
  "key" varchar(1024),
  "tag" varchar(64),
  "headerSize" integer DEFAULT 0,
  "bodySize" integer DEFAULT 0,
  "metaSize" integer DEFAULT 0,
  "expiredAt" integer DEFAULT 0,
  "staleAt" integer DEFAULT 0,
  "createdAt" integer DEFAULT 0,
  "host" varchar(128),
  "serverId" integer,
  "accessWeek" varchar(6)
);

DROP INDEX IF EXISTS "createdAt";
DROP INDEX IF EXISTS "expiredAt";
DROP INDEX IF EXISTS "serverId";

CREATE INDEX IF NOT EXISTS "staleAt"
ON "` + this.itemsTableName + `" (
  "staleAt" ASC
);

CREATE INDEX IF NOT EXISTS "hash"
ON "` + this.itemsTableName + `" (
  "hash" ASC
);

ALTER TABLE "cacheItems" ADD "accessWeek" varchar(6);
`)

		if err != nil {
			// 忽略可以预期的错误
			if strings.Contains(err.Error(), "duplicate column name") {
				err = nil
			}

			// 尝试删除重建
			if err != nil {
				if times < 3 {
					_, dropErr := this.writeDB.Exec(`DROP TABLE "` + this.itemsTableName + `"`)
					if dropErr == nil {
						return this.initTables(times + 1)
					}
					return this.WrapError(err)
				}

				return this.WrapError(err)
			}
		}
	}

	// 删除hits表
	{
		_, _ = this.writeDB.Exec(`DROP TABLE "hits"`)
	}

	return nil
}

func (this *FileListDB) listOlderItems(count int) (hashList []string, err error) {
	rows, err := this.listOlderItemsStmt.Query(count)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rows.Close()
	}()

	for rows.Next() {
		var hash string
		err = rows.Scan(&hash)
		if err != nil {
			return nil, err
		}
		hashList = append(hashList, hash)
	}

	return hashList, nil
}

func (this *FileListDB) shouldRecover() bool {
	result, err := this.writeDB.Query("pragma integrity_check;")
	if err != nil {
		logs.Println(result)
	}
	var errString = ""
	var shouldRecover = false
	if result.Next() {
		_ = result.Scan(&errString)
		if strings.TrimSpace(errString) != "ok" {
			shouldRecover = true
		}
	}
	_ = result.Close()
	return shouldRecover
}

// 删除数据库文件
func (this *FileListDB) deleteDB() {
	_ = os.Remove(this.dbPath)
	_ = os.Remove(this.dbPath + "-shm")
	_ = os.Remove(this.dbPath + "-wal")
}

// 加载Hash列表
func (this *FileListDB) loadHashMap() {
	this.hashMapIsLoaded = false

	err := this.hashMap.Load(this)
	if err != nil {
		remotelogs.Error("LIST_FILE_DB", "load hash map failed: "+err.Error()+"(file: "+this.dbPath+")")

		// 自动修复错误
		// TODO 将来希望能尽可能恢复以往数据库中的内容
		if strings.Contains(err.Error(), "database is closed") || strings.Contains(err.Error(), "database disk image is malformed") {
			_ = this.Close()
			this.deleteDB()
			remotelogs.Println("LIST_FILE_DB", "recreating the database (file:"+this.dbPath+") ...")
			err = this.Open(this.dbPath)
			if err != nil {
				remotelogs.Error("LIST_FILE_DB", "recreate the database failed: "+err.Error()+" (file:"+this.dbPath+")")
			} else {
				_ = this.Init()
			}
		}
		return
	}

	this.hashMapIsLoaded = true
}
