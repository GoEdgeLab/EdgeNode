// Copyright 2024 GoEdge CDN goedge.cdn@gmail.com. All rights reserved. Official site: https://goedge.cn .

package utils_test

import (
	"github.com/TeaOSLab/EdgeNode/internal/utils"
	"testing"
)

func TestVersionToLong(t *testing.T) {
	for _, v := range []string{
		"",
		"a",
		"1",
		"1.2",
		"1.2.1",
		"1.2.1.4",
		"1.2.3.4.5",
	} {
		t.Log(v, "=>", utils.VersionToLong(v))
	}
}
