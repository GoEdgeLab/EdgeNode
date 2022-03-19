// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package compressions

import (
	"compress/gzip"
	"github.com/TeaOSLab/EdgeNode/internal/utils"
	"io"
)

var sharedGzipWriterPool *WriterPool

func init() {
	var maxSize = utils.SystemMemoryGB() * 256
	if maxSize == 0 {
		maxSize = 256
	}
	sharedGzipWriterPool = NewWriterPool(maxSize, gzip.BestCompression, func(writer io.Writer, level int) (Writer, error) {
		return newGzipWriter(writer, level)
	})
}
