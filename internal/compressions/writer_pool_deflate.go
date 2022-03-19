// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package compressions

import (
	"compress/flate"
	"github.com/TeaOSLab/EdgeNode/internal/utils"
	"io"
)

var sharedDeflateWriterPool *WriterPool

func init() {
	var maxSize = utils.SystemMemoryGB() * 256
	if maxSize == 0 {
		maxSize = 256
	}
	sharedDeflateWriterPool = NewWriterPool(maxSize, flate.BestCompression, func(writer io.Writer, level int) (Writer, error) {
		return newDeflateWriter(writer, level)
	})
}
