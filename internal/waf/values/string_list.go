// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved. Official site: https://goedge.cn .

package values

import (
	"github.com/TeaOSLab/EdgeNode/internal/zero"
	"strings"
)

type StringList struct {
	ValueMap        map[string]zero.Zero
	CaseInsensitive bool
}

func NewStringList(caseInsensitive bool) *StringList {
	return &StringList{
		ValueMap:        map[string]zero.Zero{},
		CaseInsensitive: caseInsensitive,
	}
}

func ParseStringList(v string, caseInsensitive bool) *StringList {
	var list = NewStringList(caseInsensitive)
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
				if caseInsensitive {
					value = strings.ToLower(value)
				}
				list.ValueMap[value] = zero.Zero{}
			}
		}
	}
	return list
}

func (this *StringList) Contains(f string) bool {
	if this.CaseInsensitive {
		f = strings.ToLower(f)
	}
	_, ok := this.ValueMap[f]
	return ok
}
