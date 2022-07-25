// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package iplibrary

import (
	"database/sql"
	"github.com/TeaOSLab/EdgeCommon/pkg/rpc/pb"
	"github.com/TeaOSLab/EdgeNode/internal/events"
	"github.com/TeaOSLab/EdgeNode/internal/goman"
	"github.com/TeaOSLab/EdgeNode/internal/remotelogs"
	"github.com/iwind/TeaGo/Tea"
	_ "github.com/mattn/go-sqlite3"
	"os"
	"path/filepath"
	"time"
)

type IPListDB struct {
	db *sql.DB

	itemTableName          string
	deleteExpiredItemsStmt *sql.Stmt
	deleteItemStmt         *sql.Stmt
	insertItemStmt         *sql.Stmt
	selectItemsStmt        *sql.Stmt
	selectMaxVersionStmt   *sql.Stmt

	cleanTicker *time.Ticker

	dir string

	isClosed bool
}

func NewIPListDB() (*IPListDB, error) {
	var db = &IPListDB{
		itemTableName: "ipItems",
		dir:           filepath.Clean(Tea.Root + "/data"),
		cleanTicker:   time.NewTicker(24 * time.Hour),
	}
	err := db.init()
	return db, err
}

func (this *IPListDB) init() error {
	// 检查目录是否存在
	_, err := os.Stat(this.dir)
	if err != nil {
		err = os.MkdirAll(this.dir, 0777)
		if err != nil {
			return err
		}
		remotelogs.Println("IP_LIST_DB", "create data dir '"+this.dir+"'")
	}

	db, err := sql.Open("sqlite3", "file:"+this.dir+"/ip_list.db?cache=shared&mode=rwc&_journal_mode=WAL&_sync=OFF")
	if err != nil {
		return err
	}
	db.SetMaxOpenConns(1)

	//_, err = db.Exec("VACUUM")
	//if err != nil {
	//	return err
	//}

	this.db = db

	// 初始化数据库
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS "` + this.itemTableName + `" (
  "id" integer NOT NULL PRIMARY KEY AUTOINCREMENT,
  "listId" integer DEFAULT 0,
  "listType" varchar(32),
  "isGlobal" integer(1) DEFAULT 0,
  "type" varchar(16),
  "itemId" integer DEFAULT 0,
  "ipFrom" varchar(64) DEFAULT 0,
  "ipTo" varchar(64) DEFAULT 0,
  "expiredAt" integer DEFAULT 0,
  "eventLevel" varchar(32),
  "isDeleted" integer(1) DEFAULT 0,
  "version" integer DEFAULT 0,
  "nodeId" integer DEFAULT 0,
  "serverId" integer DEFAULT 0
);

CREATE INDEX IF NOT EXISTS "ip_list_itemId"
ON "` + this.itemTableName + `" (
  "itemId" ASC
);

CREATE INDEX IF NOT EXISTS "ip_list_expiredAt"
ON "` + this.itemTableName + `" (
  "expiredAt" ASC
);
`)
	if err != nil {
		return err
	}

	// 初始化SQL语句
	this.deleteExpiredItemsStmt, err = this.db.Prepare(`DELETE FROM "` + this.itemTableName + `" WHERE  "expiredAt">0 AND "expiredAt"<?`)
	if err != nil {
		return err
	}

	this.deleteItemStmt, err = this.db.Prepare(`DELETE FROM "` + this.itemTableName + `" WHERE "itemId"=?`)
	if err != nil {
		return err
	}

	this.insertItemStmt, err = this.db.Prepare(`INSERT INTO "` + this.itemTableName + `" ("listId", "listType", "isGlobal", "type", "itemId", "ipFrom", "ipTo", "expiredAt", "eventLevel", "isDeleted", "version", "nodeId", "serverId") VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}

	this.selectItemsStmt, err = this.db.Prepare(`SELECT "listId", "listType", "isGlobal", "type", "itemId", "ipFrom", "ipTo", "expiredAt", "eventLevel", "isDeleted", "version", "nodeId", "serverId" FROM "` + this.itemTableName + `" WHERE isDeleted=0 ORDER BY "version" ASC, "itemId" ASC LIMIT ?, ?`)
	if err != nil {
		return err
	}

	this.selectMaxVersionStmt, err = this.db.Prepare(`SELECT "version" FROM "` + this.itemTableName + `" ORDER BY "id" DESC LIMIT 1`)

	this.db = db

	goman.New(func() {
		events.On(events.EventQuit, func() {
			_ = this.Close()
			this.cleanTicker.Stop()
		})

		for range this.cleanTicker.C {
			err := this.DeleteExpiredItems()
			if err != nil {
				remotelogs.Error("IP_LIST_DB", "clean expired items failed: "+err.Error())
			}
		}
	})

	return nil
}

// DeleteExpiredItems 删除过期的条目
func (this *IPListDB) DeleteExpiredItems() error {
	if this.isClosed {
		return nil
	}

	_, err := this.deleteExpiredItemsStmt.Exec(time.Now().Unix() - 7*86400)
	return err
}

func (this *IPListDB) AddItem(item *pb.IPItem) error {
	if this.isClosed {
		return nil
	}

	_, err := this.deleteItemStmt.Exec(item.Id)
	if err != nil {
		return err
	}
	_, err = this.insertItemStmt.Exec(item.ListId, item.ListType, item.IsGlobal, item.Type, item.Id, item.IpFrom, item.IpTo, item.ExpiredAt, item.EventLevel, item.IsDeleted, item.Version, item.NodeId, item.ServerId)
	return err
}

func (this *IPListDB) ReadItems(offset int64, size int64) (items []*pb.IPItem, err error) {
	if this.isClosed {
		return
	}

	rows, err := this.selectItemsStmt.Query(offset, size)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rows.Close()
	}()

	for rows.Next() {
		//  "listId", "listType", "isGlobal", "type", "itemId", "ipFrom", "ipTo", "expiredAt", "eventLevel", "isDeleted", "version", "nodeId", "serverId"
		var pbItem = &pb.IPItem{}
		err = rows.Scan(&pbItem.ListId, &pbItem.ListType, &pbItem.IsGlobal, &pbItem.Type, &pbItem.Id, &pbItem.IpFrom, &pbItem.IpTo, &pbItem.ExpiredAt, &pbItem.EventLevel, &pbItem.IsDeleted, &pbItem.Version, &pbItem.NodeId, &pbItem.ServerId)
		if err != nil {
			return nil, err
		}
		items = append(items, pbItem)
	}
	return
}

// ReadMaxVersion 读取当前最大版本号
func (this *IPListDB) ReadMaxVersion() int64 {
	if this.isClosed {
		return 0
	}

	row := this.selectMaxVersionStmt.QueryRow()
	if row == nil {
		return 0
	}
	var version int64
	err = row.Scan(&version)
	if err != nil {
		return 0
	}
	return version
}

func (this *IPListDB) Close() error {
	this.isClosed = true

	if this.db != nil {
		_ = this.deleteExpiredItemsStmt.Close()
		_ = this.deleteItemStmt.Close()
		_ = this.insertItemStmt.Close()
		_ = this.selectItemsStmt.Close()
		_ = this.selectMaxVersionStmt.Close()

		return this.db.Close()
	}
	return nil
}
