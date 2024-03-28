// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package compressions

import (
	"compress/flate"
	teaconst "github.com/TeaOSLab/EdgeNode/internal/const"
	memutils "github.com/TeaOSLab/EdgeNode/internal/utils/mem"
	"io"
)

var sharedDeflateWriterPool *WriterPool

func init() {
	if !teaconst.IsMain {
		return
	}

	var maxSize = memutils.SystemMemoryGB() * 256
	if maxSize == 0 {
		maxSize = 256
	}
	sharedDeflateWriterPool = NewWriterPool(maxSize, flate.BestCompression, func(writer io.Writer, level int) (Writer, error) {
		return newDeflateWriter(writer, level)
	})
}
