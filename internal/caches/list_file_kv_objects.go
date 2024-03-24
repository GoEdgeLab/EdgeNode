// Copyright 2024 GoEdge CDN goedge.cdn@gmail.com. All rights reserved. Official site: https://goedge.cn .

package caches

import (
	"encoding/binary"
	"encoding/json"
	"strings"
)

// ItemKVEncoder item encoder
type ItemKVEncoder[T interface{ *Item }] struct {
}

func NewItemKVEncoder[T interface{ *Item }]() *ItemKVEncoder[T] {
	return &ItemKVEncoder[T]{}
}

func (this *ItemKVEncoder[T]) Encode(value T) ([]byte, error) {
	return json.Marshal(value)
}

func (this *ItemKVEncoder[T]) EncodeField(value T, fieldName string) ([]byte, error) {
	switch fieldName {
	case "createdAt":
		var b = make([]byte, 4)
		var createdAt = any(value).(*Item).CreatedAt
		binary.BigEndian.PutUint32(b, uint32(createdAt))
		return b, nil
	case "staleAt":
		var b = make([]byte, 4)
		var staleAt = any(value).(*Item).StaleAt
		if staleAt < 0 {
			staleAt = 0
		}
		binary.BigEndian.PutUint32(b, uint32(staleAt))
		return b, nil
	case "serverId":
		var b = make([]byte, 4)
		var serverId = any(value).(*Item).ServerId
		if serverId < 0 {
			serverId = 0
		}
		binary.BigEndian.PutUint32(b, uint32(serverId))
		return b, nil
	case "key":
		return []byte(any(value).(*Item).Key), nil
	case "wildKey":
		var key = any(value).(*Item).Key
		var dotIndex = strings.Index(key, ".")
		if dotIndex > 0 {
			var slashIndex = strings.LastIndex(key[:dotIndex], "/")
			if slashIndex > 0 {
				key = key[:dotIndex][:slashIndex+1] + "*" + key[dotIndex:]
			}
		}

		return []byte(key), nil
	}
	return nil, nil
}

func (this *ItemKVEncoder[T]) Decode(valueBytes []byte) (value T, err error) {
	err = json.Unmarshal(valueBytes, &value)
	return
}
