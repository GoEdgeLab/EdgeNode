// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved. Official site: https://goedge.cn .

package values

import (
	"github.com/TeaOSLab/EdgeNode/internal/zero"
	"github.com/iwind/TeaGo/types"
	"strings"
)

type NumberList struct {
	ValueMap map[float64]zero.Zero
}

func NewNumberList() *NumberList {
	return &NumberList{
		ValueMap: map[float64]zero.Zero{},
	}
}

func ParseNumberList(v string) *NumberList {
	var list = NewNumberList()
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
				list.ValueMap[types.Float64(value)] = zero.Zero{}
			}
		}
	}
	return list
}

func (this *NumberList) Contains(f float64) bool {
	_, ok := this.ValueMap[f]
	return ok
}
