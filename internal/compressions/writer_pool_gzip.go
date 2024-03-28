// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package compressions

import (
	"compress/gzip"
	teaconst "github.com/TeaOSLab/EdgeNode/internal/const"
	memutils "github.com/TeaOSLab/EdgeNode/internal/utils/mem"
	"io"
)

var sharedGzipWriterPool *WriterPool

func init() {
	if !teaconst.IsMain {
		return
	}

	var maxSize = memutils.SystemMemoryGB() * 256
	if maxSize == 0 {
		maxSize = 256
	}
	sharedGzipWriterPool = NewWriterPool(maxSize, gzip.BestCompression, func(writer io.Writer, level int) (Writer, error) {
		return newGzipWriter(writer, level)
	})
}
