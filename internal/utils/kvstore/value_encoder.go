// Copyright 2024 GoEdge CDN goedge.cdn@gmail.com. All rights reserved. Official site: https://goedge.cn .

package kvstore

import "encoding/json"

type ValueEncoder[T any] interface {
	Encode(value T) ([]byte, error)
	EncodeField(value T, fieldName string) ([]byte, error)
	Decode(valueBytes []byte) (value T, err error)
}

type BaseObjectEncoder[T any] struct {
}

func (this *BaseObjectEncoder[T]) Encode(value T) ([]byte, error) {
	return json.Marshal(value)
}

func (this *BaseObjectEncoder[T]) Decode(valueData []byte) (value T, err error) {
	err = json.Unmarshal(valueData, &value)
	return
}
