// Copyright 2024 GoEdge CDN goedge.cdn@gmail.com. All rights reserved. Official site: https://goedge.cn .

package utils

import (
	"github.com/TeaOSLab/EdgeNode/internal/zero"
	"strings"
)

var commonFileExtensionsMap = map[string]zero.Zero{
	".ico":   zero.New(),
	".jpg":   zero.New(),
	".jpeg":  zero.New(),
	".gif":   zero.New(),
	".png":   zero.New(),
	".webp":  zero.New(),
	".woff2": zero.New(),
	".js":    zero.New(),
	".css":   zero.New(),
	".ttf":   zero.New(),
	".otf":   zero.New(),
	".fnt":   zero.New(),
	".svg":   zero.New(),
	".map":   zero.New(),
}

// IsCommonFileExtension 判断是否为常用文件扩展名
// 不区分大小写，且不限于是否加点符号（.）
func IsCommonFileExtension(ext string) bool {
	if len(ext) == 0 {
		return false
	}
	if ext[0] != '.' {
		ext = "." + ext
	}
	_, ok := commonFileExtensionsMap[strings.ToLower(ext)]
	return ok
}
