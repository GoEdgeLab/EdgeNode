// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package compressions

import (
	teaconst "github.com/TeaOSLab/EdgeNode/internal/const"
	memutils "github.com/TeaOSLab/EdgeNode/internal/utils/mem"
	"io"
)

var sharedZSTDReaderPool *ReaderPool

func init() {
	if !teaconst.IsMain {
		return
	}

	var maxSize = memutils.SystemMemoryGB() * 256
	if maxSize == 0 {
		maxSize = 256
	}
	sharedZSTDReaderPool = NewReaderPool(maxSize, func(reader io.Reader) (Reader, error) {
		return newZSTDReader(reader)
	})
}
