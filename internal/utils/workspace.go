// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package utils

import "regexp"

var workspaceReg = regexp.MustCompile(`/Edge[A-Z]\w+/`)

func RemoveWorkspace(path string) string {
	var indexes = workspaceReg.FindAllStringIndex(path, -1)
	if len(indexes) > 0 {
		return path[indexes[len(indexes)-1][0]:]
	}
	return path
}
