// Copyright 2024 GoEdge CDN goedge.cdn@gmail.com. All rights reserved. Official site: https://goedge.cn .

package bfs

import "github.com/TeaOSLab/EdgeNode/internal/zero"

// TODO 使用atomic代替channel？需要使用基准测试对比性能
var readThreadsLimiter = make(chan zero.Zero, 16)
var writeThreadsLimiter = make(chan zero.Zero, 16)

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
