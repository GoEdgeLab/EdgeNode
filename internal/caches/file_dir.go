// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved. Official site: https://goedge.cn .

package caches

import "github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs/shared"

type FileDir struct {
	Path     string
	Capacity *shared.SizeCapacity
	IsFull   bool
}
