// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package caches

const (
	SuffixAll         = "@GOEDGE_"        // 通用后缀
	SuffixWebP        = "@GOEDGE_WEBP"    // WebP后缀
	SuffixCompression = "@GOEDGE_"        // 压缩后缀 SuffixCompression + Encoding
	SuffixMethod      = "@GOEDGE_"        // 请求方法后缀 SuffixMethod + RequestMethod
	SuffixPartial     = "@GOEDGE_partial" // 分区缓存后缀
)
