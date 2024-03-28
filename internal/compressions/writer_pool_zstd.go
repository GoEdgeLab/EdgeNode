// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package compressions

import (
	teaconst "github.com/TeaOSLab/EdgeNode/internal/const"
	memutils "github.com/TeaOSLab/EdgeNode/internal/utils/mem"
	"github.com/klauspost/compress/zstd"
	"io"
)

var sharedZSTDWriterPool *WriterPool

func init() {
	if !teaconst.IsMain {
		return
	}

	var maxSize = memutils.SystemMemoryGB() * 256
	if maxSize == 0 {
		maxSize = 256
	}
	sharedZSTDWriterPool = NewWriterPool(maxSize, int(zstd.SpeedBestCompression), func(writer io.Writer, level int) (Writer, error) {
		return newZSTDWriter(writer, level)
	})
}
