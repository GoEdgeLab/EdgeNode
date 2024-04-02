// Copyright 2024 GoEdge CDN goedge.cdn@gmail.com. All rights reserved. Official site: https://goedge.cn .

package kvstore

type NilValueEncoder[T []byte] struct {
}

func NewNilValueEncoder[T []byte]() *NilValueEncoder[T] {
	return &NilValueEncoder[T]{}
}

func (this *NilValueEncoder[T]) Encode(value T) ([]byte, error) {
	return nil, nil
}

func (this *NilValueEncoder[T]) EncodeField(value T, fieldName string) ([]byte, error) {
	return nil, nil
}

func (this *NilValueEncoder[T]) Decode(valueData []byte) (value T, err error) {
	return
}
