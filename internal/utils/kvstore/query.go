// Copyright 2024 GoEdge CDN goedge.cdn@gmail.com. All rights reserved. Official site: https://goedge.cn .

package kvstore

import (
	"bytes"
	"errors"
	"fmt"
	byteutils "github.com/TeaOSLab/EdgeNode/internal/utils/byte"
)

type DataType = int

const (
	DataTypeKey   DataType = 1
	DataTypeField DataType = 2
)

type QueryOperator int

const (
	QueryOperatorGt  QueryOperator = 1
	QueryOperatorGte QueryOperator = 2
	QueryOperatorLt  QueryOperator = 3
	QueryOperatorLte QueryOperator = 4
)

type QueryOperatorInfo struct {
	Operator QueryOperator
	Value    any
}

type IteratorFunc[T any] func(tx *Tx[T], item Item[T]) (goNext bool, err error)

type Query[T any] struct {
	table *Table[T]
	tx    *Tx[T]

	dataType  int
	offsetKey string
	limit     int
	prefix    string
	reverse   bool
	forUpdate bool

	keysOnly bool

	fieldName      string
	fieldReverse   bool
	fieldOperators []QueryOperatorInfo
	fieldPrefix    string
	fieldOffsetKey []byte
}

func NewQuery[T any]() *Query[T] {
	return &Query[T]{
		limit:    -1,
		dataType: DataTypeKey,
	}
}

func (this *Query[T]) SetTable(table *Table[T]) *Query[T] {
	this.table = table
	return this
}

func (this *Query[T]) SetTx(tx *Tx[T]) *Query[T] {
	this.tx = tx
	return this
}

func (this *Query[T]) ForKey() *Query[T] {
	this.dataType = DataTypeKey
	return this
}

func (this *Query[T]) ForField() *Query[T] {
	this.dataType = DataTypeField
	return this
}

func (this *Query[T]) Limit(limit int) *Query[T] {
	this.limit = limit
	return this
}

func (this *Query[T]) Offset(offsetKey string) *Query[T] {
	this.offsetKey = offsetKey
	return this
}

func (this *Query[T]) Prefix(prefix string) *Query[T] {
	this.prefix = prefix
	return this
}

func (this *Query[T]) Desc() *Query[T] {
	this.reverse = true
	return this
}

func (this *Query[T]) ForUpdate() *Query[T] {
	this.forUpdate = true
	return this
}

func (this *Query[T]) KeysOnly() *Query[T] {
	this.keysOnly = true
	return this
}

func (this *Query[T]) FieldAsc(fieldName string) *Query[T] {
	this.fieldName = fieldName
	this.fieldReverse = false
	return this
}

func (this *Query[T]) FieldDesc(fieldName string) *Query[T] {
	this.fieldName = fieldName
	this.fieldReverse = true
	return this
}

func (this *Query[T]) FieldPrefix(fieldName string, fieldPrefix string) *Query[T] {
	this.fieldName = fieldName
	this.fieldPrefix = fieldPrefix
	return this
}

func (this *Query[T]) FieldOffset(fieldOffset []byte) *Query[T] {
	this.fieldOffsetKey = fieldOffset
	return this
}

//func (this *Query[T]) FieldLt(value any) *Query[T] {
//	this.fieldOperators = append(this.fieldOperators, QueryOperatorInfo{
//		Operator: QueryOperatorLt,
//		Value:    value,
//	})
//	return this
//}
//
//func (this *Query[T]) FieldLte(value any) *Query[T] {
//	this.fieldOperators = append(this.fieldOperators, QueryOperatorInfo{
//		Operator: QueryOperatorLte,
//		Value:    value,
//	})
//	return this
//}
//
//func (this *Query[T]) FieldGt(value any) *Query[T] {
//	this.fieldOperators = append(this.fieldOperators, QueryOperatorInfo{
//		Operator: QueryOperatorGt,
//		Value:    value,
//	})
//	return this
//}
//
//func (this *Query[T]) FieldGte(value any) *Query[T] {
//	this.fieldOperators = append(this.fieldOperators, QueryOperatorInfo{
//		Operator: QueryOperatorGte,
//		Value:    value,
//	})
//	return this
//}

func (this *Query[T]) FindAll(fn IteratorFunc[T]) (err error) {
	defer func() {
		var panicErr = recover()
		if panicErr != nil {
			resultErr, ok := panicErr.(error)
			if ok {
				err = fmt.Errorf("execute query failed: %w", resultErr)
			}
		}
	}()

	if this.tx != nil {
		defer func() {
			_ = this.tx.Close()
		}()

		var itErr error
		if len(this.fieldName) == 0 {
			itErr = this.iterateKeys(fn)
		} else {
			itErr = this.iterateFields(fn)
		}
		if itErr != nil {
			return itErr
		}
		return this.tx.Commit()
	}

	if this.table != nil {
		var txFn func(fn func(tx *Tx[T]) error) error
		if this.forUpdate {
			txFn = this.table.WriteTx
		} else {
			txFn = this.table.ReadTx
		}

		return txFn(func(tx *Tx[T]) error {
			this.tx = tx

			if len(this.fieldName) == 0 {
				return this.iterateKeys(fn)
			}
			return this.iterateFields(fn)
		})
	}

	return errors.New("current query require 'table' or 'tx'")
}

func (this *Query[T]) iterateKeys(fn IteratorFunc[T]) error {
	if this.limit == 0 {
		return nil
	}

	var opt = &IteratorOptions{}

	var prefix []byte
	switch this.dataType {
	case DataTypeKey:
		prefix = byteutils.Append(this.table.Namespace(), []byte(KeyPrefix)...)
	case DataTypeField:
		prefix = byteutils.Append(this.table.Namespace(), []byte(FieldPrefix)...)
	default:
		prefix = byteutils.Append(this.table.Namespace(), []byte(KeyPrefix)...)
	}

	var prefixLen = len(prefix)

	if len(this.prefix) > 0 {
		prefix = append(prefix, this.prefix...)
	}

	var offsetKey []byte
	if this.reverse {
		if len(this.offsetKey) > 0 {
			offsetKey = byteutils.Append(prefix, []byte(this.offsetKey)...)
		} else {
			offsetKey = byteutils.Append(prefix, 0xFF)
		}

		opt.LowerBound = prefix
		opt.UpperBound = offsetKey
	} else {
		if len(this.offsetKey) > 0 {
			offsetKey = byteutils.Append(prefix, []byte(this.offsetKey)...)
		} else {
			offsetKey = prefix
		}
		opt.LowerBound = offsetKey
		opt.UpperBound = byteutils.Append(prefix, 0xFF)
	}

	var hasOffsetKey = len(this.offsetKey) > 0

	it, itErr := this.tx.NewIterator(opt)
	if itErr != nil {
		return itErr
	}
	defer func() {
		_ = it.Close()
	}()

	var count int

	var itemFn = func() (goNextItem bool, err error) {
		var keyBytes = it.Key()

		// skip first offset key
		if hasOffsetKey {
			hasOffsetKey = false

			if bytes.Equal(keyBytes, offsetKey) {
				return true, nil
			}
		}

		// call fn
		var value T
		if !this.keysOnly {
			valueBytes, valueErr := it.ValueAndErr()
			if valueErr != nil {
				return false, valueErr
			}
			value, err = this.table.encoder.Decode(valueBytes)
			if err != nil {
				return false, err
			}
		}

		goNext, callbackErr := fn(this.tx, Item[T]{
			Key:   string(keyBytes[prefixLen:]),
			Value: value,
		})
		if callbackErr != nil {
			if IsSkipError(callbackErr) {
				return true, nil
			} else {
				return false, callbackErr
			}
		}
		if !goNext {
			return false, nil
		}

		// limit
		if this.limit > 0 {
			count++

			if count >= this.limit {
				return false, nil
			}
		}

		return true, nil
	}

	if this.reverse {
		for it.Last(); it.Valid(); it.Prev() {
			goNext, itemErr := itemFn()
			if itemErr != nil {
				return itemErr
			}
			if !goNext {
				break
			}
		}
	} else {
		for it.First(); it.Valid(); it.Next() {
			goNext, itemErr := itemFn()
			if itemErr != nil {
				return itemErr
			}
			if !goNext {
				break
			}
		}
	}

	return nil
}

func (this *Query[T]) iterateFields(fn IteratorFunc[T]) error {
	if this.limit == 0 {
		return nil
	}

	var hasOffsetKey = len(this.offsetKey) > 0 || len(this.fieldOffsetKey) > 0

	var opt = &IteratorOptions{}

	var prefix = this.table.FieldKey(this.fieldName)
	prefix = append(prefix, '$')

	if len(this.fieldPrefix) > 0 {
		prefix = append(prefix, this.fieldPrefix...)
	}

	var offsetKey []byte
	if this.fieldReverse {
		if len(this.fieldOffsetKey) > 0 {
			offsetKey = this.fieldOffsetKey
		} else if len(this.offsetKey) > 0 {
			offsetKey = byteutils.Append(prefix, []byte(this.offsetKey)...)
		} else {
			offsetKey = byteutils.Append(prefix, 0xFF)
		}
		opt.LowerBound = prefix
		opt.UpperBound = offsetKey
	} else {
		if len(this.fieldOffsetKey) > 0 {
			offsetKey = this.fieldOffsetKey
		} else if len(this.offsetKey) > 0 {
			offsetKey = byteutils.Append(prefix, []byte(this.offsetKey)...)
			offsetKey = append(offsetKey, 0)
		} else {
			offsetKey = prefix
		}

		opt.LowerBound = offsetKey
		opt.UpperBound = byteutils.Append(prefix, 0xFF)
	}

	it, itErr := this.tx.NewIterator(opt)
	if itErr != nil {
		return itErr
	}
	defer func() {
		_ = it.Close()
	}()

	var count int

	var itemFn = func() (goNextItem bool, err error) {
		var fieldKeyBytes = it.Key()

		fieldValueBytes, keyBytes, decodeKeyErr := this.table.DecodeFieldKey(this.fieldName, fieldKeyBytes)
		if decodeKeyErr != nil {
			return false, decodeKeyErr
		}

		// skip first offset key
		if hasOffsetKey {
			hasOffsetKey = false

			if (len(this.fieldOffsetKey) > 0 && bytes.Equal(fieldKeyBytes, this.fieldOffsetKey)) ||
				bytes.Equal(fieldValueBytes, []byte(this.offsetKey)) {
				return true, nil
			}
		}

		// 执行operators
		if len(this.fieldOperators) > 0 {
			if !this.matchOperators(fieldValueBytes) {
				return true, nil
			}
		}

		var resultItem = Item[T]{
			Key:      string(keyBytes),
			FieldKey: fieldKeyBytes,
		}
		if !this.keysOnly {
			value, getErr := this.table.getWithKeyBytes(this.tx, this.table.FullKeyBytes(keyBytes))
			if getErr != nil {
				if IsNotFound(getErr) {
					return true, nil
				}
				return false, getErr
			}

			resultItem.Value = value
		}

		goNextItem, err = fn(this.tx, resultItem)
		if err != nil {
			if IsSkipError(err) {
				return true, nil
			} else {
				return false, err
			}
		}
		if !goNextItem {
			return false, nil
		}

		// limit
		if this.limit > 0 {
			count++

			if count >= this.limit {
				return false, nil
			}
		}

		return true, nil
	}

	if this.reverse {
		for it.Last(); it.Valid(); it.Prev() {
			goNext, itemErr := itemFn()
			if itemErr != nil {
				return itemErr
			}
			if !goNext {
				break
			}
		}
	} else {
		for it.First(); it.Valid(); it.Next() {
			goNext, itemErr := itemFn()
			if itemErr != nil {
				return itemErr
			}
			if !goNext {
				break
			}
		}
	}

	return nil
}

func (this *Query[T]) matchOperators(fieldValueBytes []byte) bool {
	// TODO
	return true
}
