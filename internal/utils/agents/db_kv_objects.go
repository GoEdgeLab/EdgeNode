// Copyright 2024 GoEdge CDN goedge.cdn@gmail.com. All rights reserved. Official site: https://goedge.cn .

package agents

import (
	"encoding/binary"
	"encoding/json"
	"errors"
)

type AgentIPEncoder[T interface{ *AgentIP }] struct {
}

func (this *AgentIPEncoder[T]) Encode(value T) ([]byte, error) {
	return json.Marshal(value)
}

func (this *AgentIPEncoder[T]) EncodeField(value T, fieldName string) ([]byte, error) {
	return nil, errors.New("invalid field name '" + fieldName + "'")
}

func (this *AgentIPEncoder[T]) Decode(valueBytes []byte) (value T, err error) {
	err = json.Unmarshal(valueBytes, &value)
	return
}

// EncodeKey generate key for ip item
func (this *AgentIPEncoder[T]) EncodeKey(item *AgentIP) string {
	var b = make([]byte, 8)
	if item.Id < 0 {
		item.Id = 0
	}

	binary.BigEndian.PutUint64(b, uint64(item.Id))
	return string(b)
}
