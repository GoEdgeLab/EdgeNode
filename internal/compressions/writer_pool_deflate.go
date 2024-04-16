// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package compressions

import (
	"compress/flate"
	teaconst "github.com/TeaOSLab/EdgeNode/internal/const"
	"io"
)

var sharedDeflateWriterPool *WriterPool

func init() {
	if !teaconst.IsMain {
		return
	}

	sharedDeflateWriterPool = NewWriterPool(CalculatePoolSize(), flate.BestCompression, func(writer io.Writer, level int) (Writer, error) {
		return newDeflateWriter(writer)
	})
}
