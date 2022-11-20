// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package caches

import (
	"bytes"
	"encoding/json"
	"github.com/iwind/TeaGo/types"
	"os"
	"strconv"
)

// PartialRanges 内容分区范围定义
type PartialRanges struct {
	Version  int        `json:"version"`  // 版本号
	Ranges   [][2]int64 `json:"ranges"`   // 范围
	BodySize int64      `json:"bodySize"` // 总长度
}

// NewPartialRanges 获取新对象
func NewPartialRanges(expiresAt int64) *PartialRanges {
	return &PartialRanges{
		Ranges:  [][2]int64{},
		Version: 1,
	}
}

// NewPartialRangesFromData 从数据中解析范围
func NewPartialRangesFromData(data []byte) (*PartialRanges, error) {
	var rs = NewPartialRanges(0)
	for {
		var index = bytes.IndexRune(data, '\n')
		if index < 0 {
			break
		}
		var line = data[:index]
		var colonIndex = bytes.IndexRune(line, ':')
		if colonIndex > 0 {
			switch string(line[:colonIndex]) {
			case "v": // 版本号
				rs.Version = types.Int(line[colonIndex+1:])
			case "b": // 总长度
				rs.BodySize = types.Int64(line[colonIndex+1:])
			case "r": // 范围信息
				var commaIndex = bytes.IndexRune(line, ',')
				if commaIndex > 0 {
					rs.Ranges = append(rs.Ranges, [2]int64{types.Int64(line[colonIndex+1 : commaIndex]), types.Int64(line[commaIndex+1:])})
				}
			}
		}
		data = data[index+1:]
		if len(data) == 0 {
			break
		}
	}
	return rs, nil
}

// NewPartialRangesFromJSON 从JSON中解析范围
func NewPartialRangesFromJSON(data []byte) (*PartialRanges, error) {
	var rs = NewPartialRanges(0)
	err := json.Unmarshal(data, &rs)
	if err != nil {
		return nil, err
	}
	rs.Version = 0

	return rs, nil
}

// NewPartialRangesFromFile 从文件中加载范围信息
func NewPartialRangesFromFile(path string) (*PartialRanges, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return NewPartialRanges(0), nil
	}

	// 兼容老的JSON格式
	if data[0] == '{' {
		return NewPartialRangesFromJSON(data)
	}

	// 新的格式
	return NewPartialRangesFromData(data)
}

// Add 添加新范围
func (this *PartialRanges) Add(begin int64, end int64) {
	if begin > end {
		begin, end = end, begin
	}

	var nr = [2]int64{begin, end}

	var count = len(this.Ranges)
	if count == 0 {
		this.Ranges = [][2]int64{nr}
		return
	}

	// insert
	// TODO 将来使用二分法改进
	var index = -1
	for i, r := range this.Ranges {
		if r[0] > begin || (r[0] == begin && r[1] >= end) {
			index = i
			this.Ranges = append(this.Ranges, [2]int64{})
			copy(this.Ranges[index+1:], this.Ranges[index:])
			this.Ranges[index] = nr
			break
		}
	}

	if index == -1 {
		index = count
		this.Ranges = append(this.Ranges, nr)
	}

	this.merge(index)
}

// Contains 检查是否包含某个范围
func (this *PartialRanges) Contains(begin int64, end int64) bool {
	if len(this.Ranges) == 0 {
		return false
	}

	// TODO 使用二分法查找改进性能
	for _, r2 := range this.Ranges {
		if r2[0] <= begin && r2[1] >= end {
			return true
		}
	}

	return false
}

// Nearest 查找最近的某个范围
func (this *PartialRanges) Nearest(begin int64, end int64) (r [2]int64, ok bool) {
	if len(this.Ranges) == 0 {
		return
	}

	// TODO 使用二分法查找改进性能
	for _, r2 := range this.Ranges {
		if r2[0] <= begin && r2[1] > begin {
			r = [2]int64{begin, this.min(end, r2[1])}
			ok = true
			return
		}
	}
	return
}

// 转换为字符串
func (this *PartialRanges) String() string {
	var s = "v:" + strconv.Itoa(this.Version) + "\n" + // version
		"b:" + this.formatInt64(this.BodySize) + "\n" // bodySize
	for _, r := range this.Ranges {
		s += "r:" + this.formatInt64(r[0]) + "," + this.formatInt64(r[1]) + "\n" // range
	}
	return s
}

// Bytes 将内容转换为字节
func (this *PartialRanges) Bytes() []byte {
	return []byte(this.String())
}

// WriteToFile 写入到文件中
func (this *PartialRanges) WriteToFile(path string) error {
	return os.WriteFile(path, this.Bytes(), 0666)
}

// Max 获取最大位置
func (this *PartialRanges) Max() int64 {
	if len(this.Ranges) > 0 {
		return this.Ranges[len(this.Ranges)-1][1]
	}
	return 0
}

// Reset 重置范围信息
func (this *PartialRanges) Reset() {
	this.Ranges = [][2]int64{}
}

func (this *PartialRanges) merge(index int) {
	// forward
	var lastIndex = index
	for i := index; i >= 1; i-- {
		var curr = this.Ranges[i]
		var prev = this.Ranges[i-1]
		var w1 = this.w(curr)
		var w2 = this.w(prev)
		if w1+w2 >= this.max(curr[1], prev[1])-this.min(curr[0], prev[0])-1 {
			prev = [2]int64{this.min(curr[0], prev[0]), this.max(curr[1], prev[1])}
			this.Ranges[i-1] = prev
			this.Ranges = append(this.Ranges[:i], this.Ranges[i+1:]...)
			lastIndex = i - 1
		} else {
			break
		}
	}

	// backward
	index = lastIndex
	for index < len(this.Ranges)-1 {
		var curr = this.Ranges[index]
		var next = this.Ranges[index+1]
		var w1 = this.w(curr)
		var w2 = this.w(next)
		if w1+w2 >= this.max(curr[1], next[1])-this.min(curr[0], next[0])-1 {
			curr = [2]int64{this.min(curr[0], next[0]), this.max(curr[1], next[1])}
			this.Ranges = append(this.Ranges[:index], this.Ranges[index+1:]...)
			this.Ranges[index] = curr
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

func (this *PartialRanges) formatInt64(i int64) string {
	return strconv.FormatInt(i, 10)
}
