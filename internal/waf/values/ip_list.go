// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved. Official site: https://goedge.cn .

package values

func ParseIPList(v string) *StringList {
	return ParseStringList(v, false)
}
