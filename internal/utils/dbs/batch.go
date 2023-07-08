// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved. Official site: https://goedge.cn .

package dbs

import (
	"database/sql"
	"github.com/TeaOSLab/EdgeNode/internal/remotelogs"
	"time"
)

type batchItem struct {
	query string
	args  []any
}

type Batch struct {
	db *DB
	n  int

	enableStat bool

	onFail func(err error)

	queue      chan *batchItem
	closeEvent chan bool

	isClosed bool
}

func NewBatch(db *DB, n int) *Batch {
	var batch = &Batch{
		db:         db,
		n:          n,
		queue:      make(chan *batchItem, 16),
		closeEvent: make(chan bool, 1),
	}
	db.batches = append(db.batches, batch)
	return batch
}

func (this *Batch) EnableStat(b bool) {
	this.enableStat = b
}

func (this *Batch) OnFail(callback func(err error)) {
	this.onFail = callback
}

func (this *Batch) Add(query string, args ...any) {
	if this.isClosed {
		return
	}
	this.queue <- &batchItem{
		query: query,
		args:  args,
	}
}

func (this *Batch) Exec() {
	var n = this.n
	if n <= 0 {
		n = 4
	}

	var ticker = time.NewTicker(100 * time.Millisecond)
	var count = 0
	var lastTx *sql.Tx
For:
	for {
		// closed
		if this.isClosed {
			if lastTx != nil {
				_ = this.commitTx(lastTx)
				lastTx = nil
			}

			return
		}

		select {
		case item := <-this.queue:
			if lastTx == nil {
				lastTx = this.beginTx()
				if lastTx == nil {
					continue For
				}
			}

			err := this.execItem(lastTx, item)
			if err != nil {
				if IsClosedErr(err) {
					return
				}
				this.processErr(item.query, err)
			}

			count++

			if count == n {
				count = 0
				err = this.commitTx(lastTx)
				lastTx = nil
				if err != nil {
					if IsClosedErr(err) {
						return
					}
					this.processErr("commit", err)
				}
			}
		case <-ticker.C:
			if lastTx == nil || count == 0 {
				continue For
			}
			count = 0
			err := this.commitTx(lastTx)
			lastTx = nil
			if err != nil {
				if IsClosedErr(err) {
					return
				}
				this.processErr("commit", err)
			}
		case <-this.closeEvent:
			// closed

			if lastTx != nil {
				_ = this.commitTx(lastTx)
				lastTx = nil
			}

			return
		}
	}
}

func (this *Batch) close() {
	this.isClosed = true

	select {
	case this.closeEvent <- true:
	default:

	}
}

func (this *Batch) beginTx() *sql.Tx {
	if !this.db.BeginUpdating() {
		return nil
	}

	tx, err := this.db.Begin()
	if err != nil {
		this.processErr("begin transaction", err)
		this.db.EndUpdating()
		return nil
	}
	return tx
}

func (this *Batch) commitTx(tx *sql.Tx) error {
	// always commit without checking database closing status
	this.db.EndUpdating()
	return tx.Commit()
}

func (this *Batch) execItem(tx *sql.Tx, item *batchItem) error {
	// check database status
	if this.db.BeginUpdating() {
		defer this.db.EndUpdating()
	} else {
		return errDBIsClosed
	}

	if this.enableStat {
		defer SharedQueryStatManager.AddQuery(item.query).End()
	}

	_, err := tx.Exec(item.query, item.args...)
	return err
}

func (this *Batch) processErr(prefix string, err error) {
	if err == nil {
		return
	}

	if this.onFail != nil {
		this.onFail(err)
	} else {
		remotelogs.Error("SQLITE_BATCH", prefix+": "+err.Error())
	}
}
