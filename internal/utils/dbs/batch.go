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
	db *sql.DB
	n  int

	enableStat bool

	onFail func(err error)

	queue chan *batchItem
	close chan bool

	isClosed bool
}

func NewBatch(db *sql.DB, n int) *Batch {
	return &Batch{
		db:    db,
		n:     n,
		queue: make(chan *batchItem),
		close: make(chan bool, 1),
	}
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
				_ = lastTx.Commit()
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
				this.processErr(item.query, err)
			}

			count++

			if count == n {
				count = 0
				err = lastTx.Commit()
				lastTx = nil
				if err != nil {
					this.processErr("commit", err)
				}
			}
		case <-ticker.C:
			if lastTx == nil || count == 0 {
				continue For
			}
			count = 0
			err := lastTx.Commit()
			lastTx = nil
			if err != nil {
				this.processErr("commit", err)
			}
		case <-this.close:
			// closed

			if lastTx != nil {
				_ = lastTx.Commit()
				lastTx = nil
			}

			return
		}
	}
}

func (this *Batch) Close() {
	this.isClosed = true

	select {
	case this.close <- true:
	default:

	}
}

func (this *Batch) beginTx() *sql.Tx {
	tx, err := this.db.Begin()
	if err != nil {
		this.processErr("begin transaction", err)
		return nil
	}
	return tx
}

func (this *Batch) execItem(tx *sql.Tx, item *batchItem) error {
	if this.isClosed {
		return nil
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
