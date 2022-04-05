// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved.
//go:build !plus
// +build !plus

package nodes

import (
	"os"
)

func (this *HTTPWriter) canSendfile() (*os.File, bool) {
	return nil, false
}
