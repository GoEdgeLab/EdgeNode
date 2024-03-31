// Copyright 2024 GoEdge CDN goedge.cdn@gmail.com. All rights reserved. Official site: https://goedge.cn .

package byteutils_test

import (
	"bytes"
	byteutils "github.com/TeaOSLab/EdgeNode/internal/utils/byte"
	"github.com/iwind/TeaGo/assert"
	"testing"
)

func TestCopy(t *testing.T) {
	var a = assert.NewAssertion(t)

	var prefix []byte
	prefix = append(prefix, 1, 2, 3)
	t.Log(prefix, byteutils.Copy(prefix))
	a.IsTrue(bytes.Equal(byteutils.Copy(prefix), []byte{1, 2, 3}))
}

func TestAppend(t *testing.T) {
	var as = assert.NewAssertion(t)

	var prefix []byte
	prefix = append(prefix, 1, 2, 3)

	// [1 2 3 4 5 6] [1 2 3 7]
	var a = byteutils.Append(prefix, 4, 5, 6)
	var b = byteutils.Append(prefix, 7)
	t.Log(a, b)

	as.IsTrue(bytes.Equal(a, []byte{1, 2, 3, 4, 5, 6}))
	as.IsTrue(bytes.Equal(b, []byte{1, 2, 3, 7}))
}

func TestConcat(t *testing.T) {
	var a = assert.NewAssertion(t)

	var prefix []byte
	prefix = append(prefix, 1, 2, 3)

	var b = byteutils.Contact(prefix, []byte{4, 5, 6}, []byte{7})
	t.Log(b)

	a.IsTrue(bytes.Equal(b, []byte{1, 2, 3, 4, 5, 6, 7}))
}

func TestAppend_Raw(t *testing.T) {
	var prefix []byte
	prefix = append(prefix, 1, 2, 3)

	// [1 2 3 7 5 6] [1 2 3 7]
	var a = append(prefix, 4, 5, 6)
	var b = append(prefix, 7)
	t.Log(a, b)
}
