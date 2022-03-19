// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package compressions

import (
	"github.com/TeaOSLab/EdgeNode/internal/utils"
	"io"
)

var sharedDeflateReaderPool *ReaderPool

func init() {
	var maxSize = utils.SystemMemoryGB() * 256
	if maxSize == 0 {
		maxSize = 256
	}
	sharedDeflateReaderPool = NewReaderPool(maxSize, func(reader io.Reader) (Reader, error) {
		return newDeflateReader(reader)
	})
}
