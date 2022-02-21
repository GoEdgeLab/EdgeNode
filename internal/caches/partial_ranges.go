// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package caches

import (
	"encoding/json"
)

// PartialRanges 内容分区范围定义
type PartialRanges struct {
	ranges [][2]int64
}

// NewPartialRanges 获取新对象
func NewPartialRanges() *PartialRanges {
	return &PartialRanges{ranges: [][2]int64{}}
}

// NewPartialRangesFromJSON 从JSON中解析范围
func NewPartialRangesFromJSON(data []byte) (*PartialRanges, error) {
	var rs = [][2]int64{}
	err := json.Unmarshal(data, &rs)
	if err != nil {
		return nil, err
	}
	var r = NewPartialRanges()
	r.ranges = rs
	return r, nil
}

// Add 添加新范围
func (this *PartialRanges) Add(begin int64, end int64) {
	if begin > end {
		begin, end = end, begin
	}

	var nr = [2]int64{begin, end}

	var count = len(this.ranges)
	if count == 0 {
		this.ranges = [][2]int64{nr}
		return
	}

	// insert
	// TODO 将来使用二分法改进
	var index = -1
	for i, r := range this.ranges {
		if r[0] > begin || (r[0] == begin && r[1] >= end) {
			index = i
			this.ranges = append(this.ranges, [2]int64{})
			copy(this.ranges[index+1:], this.ranges[index:])
			this.ranges[index] = nr
			break
		}
	}

	if index == -1 {
		index = count
		this.ranges = append(this.ranges, nr)
	}

	this.merge(index)
}

// Ranges 获取所有范围
func (this *PartialRanges) Ranges() [][2]int64 {
	return this.ranges
}

// Contains 检查是否包含某个范围
func (this *PartialRanges) Contains(begin int64, end int64) bool {
	if len(this.ranges) == 0 {
		return true
	}

	// TODO 使用二分法查找改进性能
	for _, r2 := range this.ranges {
		if r2[0] <= begin && r2[1] >= end {
			return true
		}
	}

	return false
}

// AsJSON 转换为JSON
func (this *PartialRanges) AsJSON() ([]byte, error) {
	return json.Marshal(this.ranges)
}

func (this *PartialRanges) merge(index int) {
	// forward
	var lastIndex = index
	for i := index; i >= 1; i-- {
		var curr = this.ranges[i]
		var prev = this.ranges[i-1]
		var w1 = this.w(curr)
		var w2 = this.w(prev)
		if w1+w2 >= this.max(curr[1], prev[1])-this.min(curr[0], prev[0])-1 {
			prev = [2]int64{this.min(curr[0], prev[0]), this.max(curr[1], prev[1])}
			this.ranges[i-1] = prev
			this.ranges = append(this.ranges[:i], this.ranges[i+1:]...)
			lastIndex = i - 1
		} else {
			break
		}
	}

	// backward
	index = lastIndex
	for index < len(this.ranges)-1 {
		var curr = this.ranges[index]
		var next = this.ranges[index+1]
		var w1 = this.w(curr)
		var w2 = this.w(next)
		if w1+w2 >= this.max(curr[1], next[1])-this.min(curr[0], next[0])-1 {
			curr = [2]int64{this.min(curr[0], next[0]), this.max(curr[1], next[1])}
			this.ranges = append(this.ranges[:index], this.ranges[index+1:]...)
			this.ranges[index] = curr
		} else {
			break
		}
	}
}

func (this *PartialRanges) w(r [2]int64) int64 {
	return r[1] - r[0]
}

func (this *PartialRanges) min(n1 int64, n2 int64) int64 {
	if n1 <= n2 {
		return n1
	}
	return n2
}

func (this *PartialRanges) max(n1 int64, n2 int64) int64 {
	if n1 >= n2 {
		return n1
	}
	return n2
}
