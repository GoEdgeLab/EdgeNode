// Copyright 2024 GoEdge CDN goedge.cdn@gmail.com. All rights reserved. Official site: https://goedge.cn .

package bfs

type BlockInfo struct {
	OriginOffsetFrom int64 `json:"1,omitempty"`
	OriginOffsetTo   int64 `json:"2,omitempty"`

	BFileOffsetFrom int64 `json:"3,omitempty"`
	BFileOffsetTo   int64 `json:"4,omitempty"`
}

func (this BlockInfo) Contains(offset int64) bool {
	return this.OriginOffsetFrom <= offset && this.OriginOffsetTo > /** MUST be gt, NOT gte **/ offset
}
