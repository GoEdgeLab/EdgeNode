// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package compressions

import (
	teaconst "github.com/TeaOSLab/EdgeNode/internal/const"
	"io"
)

var sharedZSTDWriterPool *WriterPool

func init() {
	if !teaconst.IsMain {
		return
	}

	sharedZSTDWriterPool = NewWriterPool(CalculatePoolSize(), func(writer io.Writer, level int) (Writer, error) {
		return newZSTDWriter(writer)
	})
}
