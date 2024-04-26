// Copyright 2024 GoEdge CDN goedge.cdn@gmail.com. All rights reserved. Official site: https://goedge.cn .

package bfs

import (
	"encoding/binary"
	"errors"
)

type MetaAction = byte

const (
	MetaActionNew    MetaAction = '+'
	MetaActionRemove MetaAction = '-'
)

func EncodeMetaBlock(action MetaAction, hash string, data []byte) ([]byte, error) {
	var hl = len(hash)
	if hl != HashLen {
		return nil, errors.New("invalid hash length")
	}

	var l = 1 /** Action **/ + hl /** Hash **/ + len(data)

	var b = make([]byte, 4 /** Len **/ +l)
	binary.BigEndian.PutUint32(b, uint32(l))
	b[4] = action
	copy(b[5:], hash)
	copy(b[5+hl:], data)
	return b, nil
}

func DecodeMetaBlock(blockBytes []byte) (action MetaAction, hash string, data []byte, err error) {
	var dataOffset = 4 /** Len **/ + HashLen + 1 /** Action **/
	if len(blockBytes) < dataOffset {
		err = errors.New("decode failed: invalid block data")
		return
	}

	action = blockBytes[4]
	hash = string(blockBytes[5 : 5+HashLen])

	if action == MetaActionNew {
		data = blockBytes[dataOffset:]
	}

	return
}
