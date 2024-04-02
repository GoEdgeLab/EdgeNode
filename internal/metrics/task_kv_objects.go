// Copyright 2024 GoEdge CDN goedge.cdn@gmail.com. All rights reserved. Official site: https://goedge.cn .

package metrics

import (
	"encoding/json"
	"errors"
)

type ItemEncoder[T interface{ *Stat }] struct {
}

func (this *ItemEncoder[T]) Encode(value T) ([]byte, error) {
	return json.Marshal(value)
}

func (this *ItemEncoder[T]) EncodeField(value T, fieldName string) ([]byte, error) {
	return nil, errors.New("invalid field name '" + fieldName + "'")
}

func (this *ItemEncoder[T]) Decode(valueBytes []byte) (value T, err error) {
	err = json.Unmarshal(valueBytes, &value)
	return
}
