// Copyright 2023 GoEdge CDN goedge.cdn@gmail.com. All rights reserved. Official site: https://goedge.cn .

package testutils

import (
	"fmt"
	"math/rand"
	"os"
)

// IsSingleTesting 判断当前测试环境是否为单个函数测试
func IsSingleTesting() bool {
	for _, arg := range os.Args {
		if arg == "-test.run" {
			return true
		}
	}
	return false
}

// RandIP 生成一个随机IP用于测试
func RandIP() string {
	return fmt.Sprintf("%d.%d.%d.%d", rand.Int()%255, rand.Int()%255, rand.Int()%255, rand.Int()%255)
}
