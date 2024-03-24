// Copyright 2024 GoEdge CDN goedge.cdn@gmail.com. All rights reserved. Official site: https://goedge.cn .

package kvstore

import (
	"encoding/binary"
	"github.com/iwind/TeaGo/types"
	"golang.org/x/exp/constraints"
	"strconv"
)

type IntValueEncoder[T constraints.Integer] struct {
}

func NewIntValueEncoder[T constraints.Integer]() *IntValueEncoder[T] {
	return &IntValueEncoder[T]{}
}

func (this *IntValueEncoder[T]) Encode(value T) (data []byte, err error) {
	switch v := any(value).(type) {
	case int8, int16, int32, int, uint:
		data = []byte(types.String(v))
	case int64:
		data = []byte(strconv.FormatInt(v, 16))
	case uint8:
		return []byte{v}, nil
	case uint16:
		data = make([]byte, 2)
		binary.BigEndian.PutUint16(data, v)
	case uint32:
		data = make([]byte, 4)
		binary.BigEndian.PutUint32(data, v)
	case uint64:
		data = make([]byte, 8)
		binary.BigEndian.PutUint64(data, v)
	}

	return
}

func (this *IntValueEncoder[T]) EncodeField(value T, fieldName string) ([]byte, error) {
	_ = fieldName
	return this.Encode(value)
}

func (this *IntValueEncoder[T]) Decode(valueData []byte) (value T, err error) {
	switch any(value).(type) {
	case int8:
		value = T(types.Int8(string(valueData)))
	case int16:
		value = T(types.Int16(string(valueData)))
	case int32:
		value = T(types.Int32(string(valueData)))
	case int64:
		int64Value, _ := strconv.ParseInt(string(valueData), 16, 64)
		value = T(int64Value)
	case int:
		value = T(types.Int(string(valueData)))
	case uint:
		value = T(types.Uint(string(valueData)))
	case uint8:
		if len(valueData) == 1 {
			value = T(valueData[0])
		}
	case uint16:
		if len(valueData) == 2 {
			value = T(binary.BigEndian.Uint16(valueData))
		}
	case uint32:
		if len(valueData) == 4 {
			value = T(binary.BigEndian.Uint32(valueData))
		}
	case uint64:
		if len(valueData) == 8 {
			value = T(binary.BigEndian.Uint64(valueData))
		}
	}

	return
}

