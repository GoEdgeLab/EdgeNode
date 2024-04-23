// Copyright 2024 GoEdge CDN goedge.cdn@gmail.com. All rights reserved. Official site: https://goedge.cn .

package kvstore

import (
	"errors"
	"fmt"
	"github.com/cockroachdb/pebble"
)

type Tx[T any] struct {
	table    *Table[T]
	readOnly bool

	batch *pebble.Batch
}

func NewTx[T any](table *Table[T], readOnly bool) (*Tx[T], error) {
	if table.db == nil {
		return nil, errors.New("the table has not been added to a db")
	}
	if table.db.store == nil {
		return nil, errors.New("the db has not been added to a store")
	}
	if table.db.store.rawDB == nil {
		return nil, errors.New("the store has not been opened")
	}

	return &Tx[T]{
		table:    table,
		readOnly: readOnly,
		batch:    table.db.store.rawDB.NewIndexedBatch(),
	}, nil
}

func (this *Tx[T]) Set(key string, value T) error {
	if this.readOnly {
		return errors.New("can not set value in readonly transaction")
	}

	if len(key) > KeyMaxLength {
		return ErrKeyTooLong
	}

	valueBytes, err := this.table.encoder.Encode(value)
	if err != nil {
		return err
	}

	return this.table.set(this, key, valueBytes, value, false, false)
}

func (this *Tx[T]) SetSync(key string, value T) error {
	if this.readOnly {
		return errors.New("can not set value in readonly transaction")
	}

	if len(key) > KeyMaxLength {
		return ErrKeyTooLong
	}

	valueBytes, err := this.table.encoder.Encode(value)
	if err != nil {
		return err
	}

	return this.table.set(this, key, valueBytes, value, false, true)
}

func (this *Tx[T]) Insert(key string, value T) error {
	if this.readOnly {
		return errors.New("can not set value in readonly transaction")
	}

	if len(key) > KeyMaxLength {
		return ErrKeyTooLong
	}

	valueBytes, err := this.table.encoder.Encode(value)
	if err != nil {
		return err
	}

	return this.table.set(this, key, valueBytes, value, true, false)
}

func (this *Tx[T]) Get(key string) (value T, err error) {
	if this.table.isClosed {
		err = NewTableClosedErr(this.table.name)
		return
	}
	return this.table.get(this, key)
}

func (this *Tx[T]) Delete(key string) error {
	if this.table.isClosed {
		return NewTableClosedErr(this.table.name)
	}
	if this.readOnly {
		return errors.New("can not delete value in readonly transaction")
	}

	return this.table.deleteKeys(this, key)
}

func (this *Tx[T]) NewIterator(opt *IteratorOptions) (*pebble.Iterator, error) {
	return this.batch.NewIter(opt.RawOptions())
}

func (this *Tx[T]) Close() error {
	return this.batch.Close()
}

func (this *Tx[T]) Commit() (err error) {
	return this.commit(DefaultWriteOptions)
}

func (this *Tx[T]) CommitSync() (err error) {
	return this.commit(DefaultWriteSyncOptions)
}

func (this *Tx[T]) Query() *Query[T] {
	var query = NewQuery[T]()
	query.SetTx(this)
	return query
}

func (this *Tx[T]) commit(opt *pebble.WriteOptions) (err error) {
	defer func() {
		var panicErr = recover()
		if panicErr != nil {
			resultErr, ok := panicErr.(error)
			if ok {
				err = fmt.Errorf("commit batch failed: %w", resultErr)
			}
		}
	}()

	return this.batch.Commit(opt)
}
