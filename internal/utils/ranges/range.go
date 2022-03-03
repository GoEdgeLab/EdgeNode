// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package rangeutils

import "strconv"

type Range [2]int64

func NewRange(start int64, end int64) Range {
	return [2]int64{start, end}
}

func (this Range) Start() int64 {
	return this[0]
}

func (this Range) End() int64 {
	return this[1]
}

func (this Range) Length() int64 {
	return this[1] - this[0] + 1
}

func (this Range) Convert(total int64) (newRange Range, ok bool) {
	if total <= 0 {
		return this, false
	}
	if this[0] < 0 {
		this[0] += total
		if this[0] < 0 {
			return this, false
		}
		this[1] = total - 1
	}
	if this[1] < 0 {
		this[1] = total - 1
	}
	if this[1] > total-1 {
		this[1] = total - 1
	}
	if this[0] > this[1] {
		return this, false
	}

	return this, true
}

// ComposeContentRangeHeader 组合Content-Range Header
// totalSize 可能是一个数字，也可能是一个星号（*）
func (this Range) ComposeContentRangeHeader(totalSize string) string {
	return "bytes " + strconv.FormatInt(this[0], 10) + "-" + strconv.FormatInt(this[1], 10) + "/" + totalSize
}
