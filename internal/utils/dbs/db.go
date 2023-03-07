// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package dbs

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/TeaOSLab/EdgeNode/internal/events"
	"github.com/TeaOSLab/EdgeNode/internal/remotelogs"
	"github.com/TeaOSLab/EdgeNode/internal/utils/fileutils"
	_ "github.com/mattn/go-sqlite3"
	"strings"
)

type DB struct {
	locker *fileutils.Locker
	rawDB  *sql.DB

	enableStat bool
}

func OpenWriter(dsn string) (*DB, error) {
	return open(dsn, true)
}

func OpenReader(dsn string) (*DB, error) {
	return open(dsn, false)
}

func open(dsn string, lock bool) (*DB, error) {
	// locker
	var locker *fileutils.Locker
	if lock {
		var path = dsn
		var queryIndex = strings.Index(dsn, "?")
		if queryIndex >= 0 {
			path = path[:queryIndex]
		}
		path = strings.TrimSpace(strings.TrimPrefix(path, "file:"))
		locker = fileutils.NewLocker(path)
		err := locker.Lock()
		if err != nil {
			remotelogs.Warn("DB", "lock '"+path+"' failed: "+err.Error())
			locker = nil
		}
	}

	// open
	rawDB, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, err
	}

	var db = NewDB(rawDB)
	db.locker = locker
	return db, nil
}

func NewDB(rawDB *sql.DB) *DB {
	var db = &DB{
		rawDB: rawDB,
	}

	events.OnKey(events.EventQuit, fmt.Sprintf("db_%p", db), func() {
		_ = rawDB.Close()
	})
	events.OnKey(events.EventTerminated, fmt.Sprintf("db_%p", db), func() {
		_ = rawDB.Close()
	})

	return db
}

func (this *DB) SetMaxOpenConns(n int) {
	this.rawDB.SetMaxOpenConns(n)
}

func (this *DB) EnableStat(b bool) {
	this.enableStat = b
}

func (this *DB) Begin() (*sql.Tx, error) {
	return this.rawDB.Begin()
}

func (this *DB) Prepare(query string) (*Stmt, error) {
	stmt, err := this.rawDB.Prepare(query)
	if err != nil {
		return nil, err
	}

	var s = NewStmt(stmt, query)
	if this.enableStat {
		s.EnableStat()
	}
	return s, nil
}

func (this *DB) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	if this.enableStat {
		defer SharedQueryStatManager.AddQuery(query).End()
	}
	return this.rawDB.ExecContext(ctx, query, args...)
}

func (this *DB) Exec(query string, args ...interface{}) (sql.Result, error) {
	if this.enableStat {
		defer SharedQueryStatManager.AddQuery(query).End()
	}
	return this.rawDB.Exec(query, args...)
}

func (this *DB) Query(query string, args ...interface{}) (*sql.Rows, error) {
	if this.enableStat {
		defer SharedQueryStatManager.AddQuery(query).End()
	}
	return this.rawDB.Query(query, args...)
}

func (this *DB) QueryRow(query string, args ...interface{}) *sql.Row {
	if this.enableStat {
		defer SharedQueryStatManager.AddQuery(query).End()
	}
	return this.rawDB.QueryRow(query, args...)
}

func (this *DB) Close() error {
	events.Remove(fmt.Sprintf("db_%p", this))

	defer func() {
		if this.locker != nil {
			_ = this.locker.Release()
		}
	}()

	return this.rawDB.Close()
}

func (this *DB) RawDB() *sql.DB {
	return this.rawDB
}
