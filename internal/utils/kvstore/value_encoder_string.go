// Copyright 2024 GoEdge CDN goedge.cdn@gmail.com. All rights reserved. Official site: https://goedge.cn .

package kvstore

type StringValueEncoder[T string] struct {
}

func NewStringValueEncoder[T string]() *StringValueEncoder[T] {
	return &StringValueEncoder[T]{}
}

func (this *StringValueEncoder[T]) Encode(value T) ([]byte, error) {
	return []byte(value), nil
}

func (this *StringValueEncoder[T]) EncodeField(value T, fieldName string) ([]byte, error) {
	_ = fieldName
	return this.Encode(value)
}

func (this *StringValueEncoder[T]) Decode(valueData []byte) (value T, err error) {
	value = T(valueData)
	return
}
