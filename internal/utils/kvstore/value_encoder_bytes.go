// Copyright 2024 GoEdge CDN goedge.cdn@gmail.com. All rights reserved. Official site: https://goedge.cn .

package kvstore

type BytesValueEncoder[T []byte] struct {
}

func NewBytesValueEncoder[T []byte]() *BytesValueEncoder[T] {
	return &BytesValueEncoder[T]{}
}

func (this *BytesValueEncoder[T]) Encode(value T) ([]byte, error) {
	return value, nil
}

func (this *BytesValueEncoder[T]) EncodeField(value T, fieldName string) ([]byte, error) {
	_ = fieldName
	return this.Encode(value)
}

func (this *BytesValueEncoder[T]) Decode(valueData []byte) (value T, err error) {
	value = valueData
	return
}
