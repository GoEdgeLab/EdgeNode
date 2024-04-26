// Copyright 2024 GoEdge CDN goedge.cdn@gmail.com. All rights reserved. Official site: https://goedge.cn .

package bfs

import (
	"errors"
	"os"
)

var ErrClosed = errors.New("the file closed")
var ErrInvalidHash = errors.New("invalid hash")
var ErrFileIsWriting = errors.New("the file is writing")

func IsWritingErr(err error) bool {
	return err != nil && errors.Is(err, ErrFileIsWriting)
}

func IsNotExist(err error) bool {
	return err != nil && os.IsNotExist(err)
}
