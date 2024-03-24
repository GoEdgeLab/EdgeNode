// Copyright 2024 GoEdge CDN goedge.cdn@gmail.com. All rights reserved. Official site: https://goedge.cn .

package kvstore

type Item[T any] struct {
	Key      string
	Value    T
	FieldKey []byte
}
