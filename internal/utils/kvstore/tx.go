// Copyright 2024 GoEdge CDN goedge.cdn@gmail.com. All rights reserved. Official site: https://goedge.cn .

package kvstore

import (
	"errors"
	"github.com/cockroachdb/pebble"
)

type Tx[T any] struct {
	table    *Table[T]
	readOnly bool

	batch *pebble.Batch
}

func NewTx[T any](table *Table[T], readOnly bool) *Tx[T] {
	return &Tx[T]{
		table:    table,
		readOnly: readOnly,
		batch:    table.db.store.rawDB.NewIndexedBatch(),
	}
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

	return this.table.set(this, key, valueBytes, value)
}

func (this *Tx[T]) Get(key string) (value T, err error) {
	return this.table.get(this, key)
}

func (this *Tx[T]) Delete(key string) error {
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

func (this *Tx[T]) Commit() error {
	return this.batch.Commit(DefaultWriteOptions)
}

func (this *Tx[T]) Query() *Query[T] {
	var query = NewQuery[T]()
	query.SetTx(this)
	return query
}
