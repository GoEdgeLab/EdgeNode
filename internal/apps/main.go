// Copyright 2023 Liuxiangchao iwind.liu@gmail.com. All rights reserved. Official site: https://goedge.cn .

package apps

import teaconst "github.com/TeaOSLab/EdgeNode/internal/const"

func RunMain(f func()) {
	if teaconst.IsMain {
		f()
	}
}
