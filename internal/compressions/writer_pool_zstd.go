// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package compressions

import (
	"github.com/TeaOSLab/EdgeNode/internal/utils"
	"github.com/klauspost/compress/zstd"
	"io"
)

var sharedZSTDWriterPool *WriterPool

func init() {
	var maxSize = utils.SystemMemoryGB() * 256
	if maxSize == 0 {
		maxSize = 256
	}
	sharedZSTDWriterPool = NewWriterPool(maxSize, int(zstd.SpeedBestCompression), func(writer io.Writer, level int) (Writer, error) {
		return newZSTDWriter(writer, level)
	})
}
