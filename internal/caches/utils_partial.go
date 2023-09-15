// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package caches

import "strings"

// PartialRangesFilePath 获取 ranges 文件路径
func PartialRangesFilePath(path string) string {
	// ranges路径
	var dotIndex = strings.LastIndex(path, ".")
	var rangePath string
	if dotIndex < 0 {
		rangePath = path + "@ranges.cache"
	} else {
		rangePath = path[:dotIndex] + "@ranges" + path[dotIndex:]
	}
	return rangePath
}
