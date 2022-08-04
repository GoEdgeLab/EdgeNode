// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.
//go:build !arm64
// +build !arm64

package nodes

import (
	"bytes"
	"github.com/TeaOSLab/EdgeNode/internal/remotelogs"
	"github.com/iwind/TeaGo/Tea"
	"github.com/iwind/TeaGo/logs"
	"os"
	"syscall"
)

// 处理异常
func (this *Node) handlePanic() {
	// 如果是在前台运行就直接返回
	backgroundEnv, _ := os.LookupEnv("EdgeBackground")
	if backgroundEnv != "on" {
		return
	}

	var panicFile = Tea.Root + "/logs/panic.log"

	// 分析panic
	data, err := os.ReadFile(panicFile)
	if err == nil {
		var index = bytes.Index(data, []byte("panic:"))
		if index >= 0 {
			remotelogs.Error("NODE", "系统错误，请上报给开发者: "+string(data[index:]))
		}
	}

	fp, err := os.OpenFile(panicFile, os.O_CREATE|os.O_TRUNC|os.O_WRONLY|os.O_APPEND, 0777)
	if err != nil {
		logs.Println("NODE", "open 'panic.log' failed: "+err.Error())
		return
	}
	err = syscall.Dup2(int(fp.Fd()), int(os.Stderr.Fd()))
	if err != nil {
		logs.Println("NODE", "write to 'panic.log' failed: "+err.Error())
	}
}

