// Copyright 2023 GoEdge CDN goedge.cdn@gmail.com. All rights reserved. Official site: https://goedge.cn .

package counters

type Span struct {
	Timestamp int64
	Count     uint64
}

func NewSpan() *Span {
	return &Span{}
}
