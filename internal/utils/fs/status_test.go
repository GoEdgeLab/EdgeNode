// Copyright 2023 GoEdge CDN goedge.cdn@gmail.com. All rights reserved. Official site: https://goedge.cn .

package fsutils_test

import (
	fsutils "github.com/TeaOSLab/EdgeNode/internal/utils/fs"
	"testing"
	"time"
)

func TestWaitLoad(t *testing.T) {
	fsutils.WaitLoad(100, 5, 1*time.Minute)
}
