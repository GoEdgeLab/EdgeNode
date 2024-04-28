// Copyright 2024 GoEdge CDN goedge.cdn@gmail.com. All rights reserved. Official site: https://goedge.cn .

package percpu

import (
	_ "unsafe"
)

//go:linkname runtime_procPin runtime.procPin
func runtime_procPin() int

//go:linkname runtime_procUnpin runtime.procUnpin
func runtime_procUnpin() int

func GetProcId() int {
	var pid = runtime_procPin()
	runtime_procUnpin()
	return pid
}
