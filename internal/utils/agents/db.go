// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package agents

import (
	"errors"
	"github.com/TeaOSLab/EdgeNode/internal/events"
	"github.com/TeaOSLab/EdgeNode/internal/remotelogs"
	"github.com/TeaOSLab/EdgeNode/internal/utils/dbs"
	"github.com/iwind/TeaGo/Tea"
	"github.com/iwind/TeaGo/types"
	"log"
	"os"
	"path/filepath"
	"strings"
)

const (
	tableAgentIPs = "agentIPs"
)

type DB struct {
	db   *dbs.DB
	path string

	insertAgentIPStmt *dbs.Stmt
	listAgentIPsStmt  *dbs.Stmt
}

func NewDB(path string) *DB {
	var db = &DB{path: path}

	events.OnClose(func() {
		_ = db.Close()
	})

	return db
}

func (this *DB) Init() error {
	// 检查目录是否存在
	var dir = filepath.Dir(this.path)

	_, err := os.Stat(dir)
	if err != nil {
		err = os.MkdirAll(dir, 0777)
		if err != nil {
			return err
		}
		remotelogs.Println("DB", "create database dir '"+dir+"'")
	}

	// TODO 思考 data.db 的数据安全性
	db, err := dbs.OpenWriter("file:" + this.path + "?cache=shared&mode=rwc&_journal_mode=WAL&_locking_mode=EXCLUSIVE")
	if err != nil {
		return err
	}
	db.SetMaxOpenConns(1)

	/**_, err = db.Exec("VACUUM")
	if err != nil {
		return err
	}**/

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS "` + tableAgentIPs + `" (
  "id" integer NOT NULL PRIMARY KEY AUTOINCREMENT,
  "ip" varchar(64),
  "agentCode" varchar(128)
);`)
	if err != nil {
		return err
	}

	// 预编译语句

	// agent ip record statements
	this.insertAgentIPStmt, err = db.Prepare(`INSERT INTO "` + tableAgentIPs + `" ("id", "ip", "agentCode") VALUES (?, ?, ?)`)
	if err != nil {
		return err
	}

	this.listAgentIPsStmt, err = db.Prepare(`SELECT "id", "ip", "agentCode" FROM "` + tableAgentIPs + `" ORDER BY "id" ASC LIMIT ? OFFSET ?`)
	if err != nil {
		return err
	}

	this.db = db

	return nil
}

func (this *DB) InsertAgentIP(ipId int64, ip string, agentCode string) error {
	if this.db == nil {
		return errors.New("db should not be nil")
	}

	_, err := this.insertAgentIPStmt.Exec(ipId, ip, agentCode)
	if err != nil {
		// 不提示ID重复错误
		if strings.Contains(err.Error(), "UNIQUE constraint") {
			return nil
		}

		return err
	}

	return nil
}

func (this *DB) ListAgentIPs(offset int64, size int64) (agentIPs []*AgentIP, err error) {
	if this.db == nil {
		return nil, errors.New("db should not be nil")
	}
	rows, err := this.listAgentIPsStmt.Query(size, offset)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rows.Close()
	}()
	for rows.Next() {
		var agentIP = &AgentIP{}
		err = rows.Scan(&agentIP.Id, &agentIP.IP, &agentIP.AgentCode)
		if err != nil {
			return nil, err
		}
		agentIPs = append(agentIPs, agentIP)
	}
	return
}

func (this *DB) Close() error {
	if this.db == nil {
		return nil
	}

	for _, stmt := range []*dbs.Stmt{
		this.insertAgentIPStmt,
		this.listAgentIPsStmt,
	} {
		if stmt != nil {
			_ = stmt.Close()
		}
	}

	return this.db.Close()
}

// 打印日志
func (this *DB) log(args ...any) {
	if !Tea.IsTesting() {
		return
	}
	if len(args) == 0 {
		return
	}

	args[0] = "[" + types.String(args[0]) + "]"
	log.Println(args...)
}
