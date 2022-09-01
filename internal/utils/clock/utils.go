// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved. Official site: https://goedge.cn .

package clock

import (
	"bytes"
	"errors"
	"github.com/TeaOSLab/EdgeNode/internal/remotelogs"
	"os/exec"
	"runtime"
	"time"
)

// Start TODO 需要可以在集群中配置
func Start() {
	// sync once
	err := Sync()
	if err != nil {
		remotelogs.Warn("CLOCK", "sync time clock failed: "+err.Error())
	}

	var ticker = time.NewTicker(1 * time.Hour)
	for range ticker.C {
		err := Sync()
		if err != nil {
			// ignore error
		}
	}
}

// Sync 自动校对时间
func Sync() error {
	if runtime.GOOS != "linux" {
		return nil
	}

	ntpdate, err := exec.LookPath("ntpdate")
	if err != nil {
		return nil
	}
	if len(ntpdate) > 0 {
		return syncNtpdate(ntpdate)
	}

	return nil
}

func syncNtpdate(ntpdate string) error {
	var cmd = exec.Command(ntpdate, "pool.ntp.org")
	var stderr = &bytes.Buffer{}
	cmd.Stderr = stderr
	err := cmd.Run()
	if err != nil {
		return errors.New(err.Error() + ": " + stderr.String())
	}

	return nil
}
