// Copyright 2024 GoEdge CDN goedge.cdn@gmail.com. All rights reserved. Official site: https://goedge.cn .

package iplibrary

import (
	"encoding/binary"
	"errors"
	"github.com/TeaOSLab/EdgeCommon/pkg/rpc/pb"
	"google.golang.org/protobuf/proto"
	"math"
)

type IPItemEncoder[T interface{ *pb.IPItem }] struct {
}

func NewIPItemEncoder[T interface{ *pb.IPItem }]() *IPItemEncoder[T] {
	return &IPItemEncoder[T]{}
}

func (this *IPItemEncoder[T]) Encode(value T) ([]byte, error) {
	return proto.Marshal(any(value).(*pb.IPItem))
}

func (this *IPItemEncoder[T]) EncodeField(value T, fieldName string) ([]byte, error) {
	switch fieldName {
	case "expiresAt":
		var expiresAt = any(value).(*pb.IPItem).ExpiredAt
		if expiresAt < 0 || expiresAt > int64(math.MaxUint32) {
			expiresAt = 0
		}
		var b = make([]byte, 4)
		binary.BigEndian.PutUint32(b, uint32(expiresAt))
		return b, nil
	}

	return nil, errors.New("field '" + fieldName + "' not found")
}

func (this *IPItemEncoder[T]) Decode(valueBytes []byte) (value T, err error) {
	var item = &pb.IPItem{}
	err = proto.Unmarshal(valueBytes, item)
	value = item
	return
}

// EncodeKey generate key for ip item
func (this *IPItemEncoder[T]) EncodeKey(item *pb.IPItem) string {
	var b = make([]byte, 8)
	if item.Id < 0 {
		item.Id = 0
	}

	binary.BigEndian.PutUint64(b, uint64(item.Id))
	return string(b)
}
