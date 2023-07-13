// Copyright 2023 GoEdge CDN goedge.cdn@gmail.com. All rights reserved. Official site: https://goedge.cn .

package waf

import "github.com/TeaOSLab/EdgeNode/internal/utils/counters"

var SharedCounter = counters.NewCounter().WithGC()
