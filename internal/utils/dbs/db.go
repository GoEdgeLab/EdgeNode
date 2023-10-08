// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package dbs

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	teaconst "github.com/TeaOSLab/EdgeNode/internal/const"
	"github.com/TeaOSLab/EdgeNode/internal/events"
	"github.com/TeaOSLab/EdgeNode/internal/remotelogs"
	"github.com/TeaOSLab/EdgeNode/internal/utils/fs"
	_ "github.com/mattn/go-sqlite3"
	"os"
	"strings"
	"sync"
	"time"
)

const (
	SyncMode = "OFF"
)

var errDBIsClosed = errors.New("the database is closed")

type DB struct {
	locker *fsutils.Locker
	rawDB  *sql.DB
	dsn    string

	statusLocker  sync.Mutex
	countUpdating int32

	isClosing bool

	enableStat bool

	batches []*Batch
}

func OpenWriter(dsn string) (*DB, error) {
	return open(dsn, true)
}

func OpenReader(dsn string) (*DB, error) {
	return open(dsn, false)
}

func open(dsn string, lock bool) (*DB, error) {
	if teaconst.IsQuiting {
		return nil, errors.New("can not open database when process is quiting")
	}

	// decode path
	var path = dsn
	var queryIndex = strings.Index(dsn, "?")
	if queryIndex >= 0 {
		path = path[:queryIndex]
	}
	path = strings.TrimSpace(strings.TrimPrefix(path, "file:"))

	// locker
	var locker *fsutils.Locker
	if lock {
		locker = fsutils.NewLocker(path)
		err := locker.Lock()
		if err != nil {
			remotelogs.Warn("DB", "lock '"+path+"' failed: "+err.Error())
			locker = nil
		}
	}

	// check if closed successfully last time, if not we recover it
	var walPath = path + "-wal"
	_, statWalErr := os.Stat(walPath)
	var shouldRecover = statWalErr == nil

	// open
	rawDB, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, err
	}

	if shouldRecover {
		err = rawDB.Close()
		if err != nil {
			return nil, err
		}

		// open again
		rawDB, err = sql.Open("sqlite3", dsn)
		if err != nil {
			return nil, err
		}
	}

	var db = NewDB(rawDB, dsn)
	db.locker = locker
	return db, nil
}

func NewDB(rawDB *sql.DB, dsn string) *DB {
	var db = &DB{
		rawDB: rawDB,
		dsn:   dsn,
	}

	events.OnKey(events.EventQuit, fmt.Sprintf("db_%p", db), func() {
		_ = db.Close()
	})
	events.OnKey(events.EventTerminated, fmt.Sprintf("db_%p", db), func() {
		_ = db.Close()
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
	// check database status
	if this.BeginUpdating() {
		defer this.EndUpdating()
	} else {
		return nil, errDBIsClosed
	}

	return this.rawDB.Begin()
}

func (this *DB) Prepare(query string) (*Stmt, error) {
	stmt, err := this.rawDB.Prepare(query)
	if err != nil {
		return nil, err
	}

	var s = NewStmt(this, stmt, query)
	if this.enableStat {
		s.EnableStat()
	}
	return s, nil
}

func (this *DB) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	// check database status
	if this.BeginUpdating() {
		defer this.EndUpdating()
	} else {
		return nil, errDBIsClosed
	}

	if this.enableStat {
		defer SharedQueryStatManager.AddQuery(query).End()
	}

	return this.rawDB.ExecContext(ctx, query, args...)
}

func (this *DB) Exec(query string, args ...any) (sql.Result, error) {
	// check database status
	if this.BeginUpdating() {
		defer this.EndUpdating()
	} else {
		return nil, errDBIsClosed
	}

	if this.enableStat {
		defer SharedQueryStatManager.AddQuery(query).End()
	}
	return this.rawDB.Exec(query, args...)
}

func (this *DB) Query(query string, args ...any) (*sql.Rows, error) {
	if this.enableStat {
		defer SharedQueryStatManager.AddQuery(query).End()
	}
	return this.rawDB.Query(query, args...)
}

func (this *DB) QueryRow(query string, args ...any) *sql.Row {
	if this.enableStat {
		defer SharedQueryStatManager.AddQuery(query).End()
	}
	return this.rawDB.QueryRow(query, args...)
}

// Close the database
func (this *DB) Close() error {
	// check database status
	this.statusLocker.Lock()
	if this.isClosing {
		this.statusLocker.Unlock()
		return nil
	}
	this.isClosing = true
	this.statusLocker.Unlock()

	// waiting for updating operations to finish
	var maxLoops = 5_000
	for {
		this.statusLocker.Lock()
		var countUpdating = this.countUpdating
		this.statusLocker.Unlock()
		if countUpdating <= 0 {
			break
		}
		time.Sleep(1 * time.Millisecond)

		maxLoops--
		if maxLoops <= 0 {
			break
		}
	}

	for _, batch := range this.batches {
		batch.close()
	}

	events.Remove(fmt.Sprintf("db_%p", this))

	defer func() {
		if this.locker != nil {
			_ = this.locker.Release()
		}
	}()

	// print log
	/**if len(this.dsn) > 0 {
		u, _ := url.Parse(this.dsn)
		if u != nil && len(u.Path) > 0 {
			remotelogs.Debug("DB", "close '"+u.Path)
		}
	}**/

	return this.rawDB.Close()
}

func (this *DB) BeginUpdating() bool {
	this.statusLocker.Lock()
	defer this.statusLocker.Unlock()

	if this.isClosing {
		return false
	}

	this.countUpdating++
	return true
}

func (this *DB) EndUpdating() {
	this.statusLocker.Lock()
	this.countUpdating--
	this.statusLocker.Unlock()
}

func (this *DB) RawDB() *sql.DB {
	return this.rawDB
}
