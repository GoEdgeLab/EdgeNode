// Copyright 2024 GoEdge CDN goedge.cdn@gmail.com. All rights reserved. Official site: https://goedge.cn .

package kvstore

type BoolValueEncoder[T bool] struct {
}

func NewBoolValueEncoder[T bool]() *BoolValueEncoder[T] {
	return &BoolValueEncoder[T]{}
}

func (this *BoolValueEncoder[T]) Encode(value T) ([]byte, error) {
	if value {
		return []byte{1}, nil
	}
	return []byte{0}, nil
}

func (this *BoolValueEncoder[T]) EncodeField(value T, fieldName string) ([]byte, error) {
	_ = fieldName
	return this.Encode(value)
}

func (this *BoolValueEncoder[T]) Decode(valueData []byte) (value T, err error) {
	if len(valueData) == 1 {
		value = valueData[0] == 1
	}
	return
}
