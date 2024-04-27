// Copyright 2024 GoEdge CDN goedge.cdn@gmail.com. All rights reserved. Official site: https://goedge.cn .

package bfs

import "github.com/TeaOSLab/EdgeNode/internal/zero"

// TODO 使用atomic代替channel？需要使用基准测试对比性能
// TODO 线程数可以根据硬盘数量动态调整？
var readThreadsLimiter = make(chan zero.Zero, 8)
var writeThreadsLimiter = make(chan zero.Zero, 8)

func AckReadThread() {
	readThreadsLimiter <- zero.Zero{}
}

func ReleaseReadThread() {
	<-readThreadsLimiter
}

func AckWriteThread() {
	writeThreadsLimiter <- zero.Zero{}
}

func ReleaseWriteThread() {
	<-writeThreadsLimiter
}
