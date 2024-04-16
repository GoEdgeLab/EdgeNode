// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package compressions

import (
	"compress/gzip"
	teaconst "github.com/TeaOSLab/EdgeNode/internal/const"
	"io"
)

var sharedGzipWriterPool *WriterPool

func init() {
	if !teaconst.IsMain {
		return
	}


	sharedGzipWriterPool = NewWriterPool(CalculatePoolSize(), gzip.BestCompression, func(writer io.Writer, level int) (Writer, error) {
		return newGzipWriter(writer)
	})
}
