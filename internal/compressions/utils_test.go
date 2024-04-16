// Copyright 2024 GoEdge CDN goedge.cdn@gmail.com. All rights reserved. Official site: https://goedge.cn .

package compressions_test

import (
	"github.com/TeaOSLab/EdgeNode/internal/compressions"
	"github.com/iwind/TeaGo/assert"
	"testing"
)

func TestGenerateCompressLevel(t *testing.T) {
	var a = assert.NewAssertion(t)

	t.Log(compressions.GenerateCompressLevel(0, 10))
	t.Log(compressions.GenerateCompressLevel(1, 10))
	t.Log(compressions.GenerateCompressLevel(1, 4))

	{
		var level = compressions.GenerateCompressLevel(1, 2)
		t.Log(level)
		a.IsTrue(level >= 1 && level <= 2)
	}
}

func TestCalculatePoolSize(t *testing.T) {
	t.Log(compressions.CalculatePoolSize())
}
