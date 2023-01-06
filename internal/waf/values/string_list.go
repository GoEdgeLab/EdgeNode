// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved. Official site: https://goedge.cn .

package values

import (
	"github.com/TeaOSLab/EdgeNode/internal/zero"
	"strings"
)

type StringList struct {
	ValueMap map[string]zero.Zero
}

func NewStringList() *StringList {
	return &StringList{
		ValueMap: map[string]zero.Zero{},
	}
}

func ParseStringList(v string) *StringList {
	var list = NewStringList()
	if len(v) == 0 {
		return list
	}

	var lines = strings.Split(v, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if len(line) == 0 {
			continue
		}

		var values = strings.Split(line, ",")
		for _, value := range values {
			value = strings.TrimSpace(value)
			if len(value) > 0 {
				list.ValueMap[value] = zero.Zero{}
			}
		}
	}
	return list
}

func (this *StringList) Contains(f string) bool {
	_, ok := this.ValueMap[f]
	return ok
}
