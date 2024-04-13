// Copyright 2024 GoEdge CDN goedge.cdn@gmail.com. All rights reserved. Official site: https://goedge.cn .

package utils_test

import (
	"github.com/TeaOSLab/EdgeNode/internal/utils"
	"github.com/iwind/TeaGo/assert"
	"testing"
)

func TestIsCommonFileExtension(t *testing.T) {
	var a = assert.NewAssertion(t)

	a.IsTrue(utils.IsCommonFileExtension(".jpg"))
	a.IsTrue(utils.IsCommonFileExtension("png"))
	a.IsTrue(utils.IsCommonFileExtension("PNG"))
	a.IsTrue(utils.IsCommonFileExtension(".PNG"))
	a.IsTrue(utils.IsCommonFileExtension("Png"))
	a.IsFalse(utils.IsCommonFileExtension("zip"))
}
