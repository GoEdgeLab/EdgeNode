// Copyright 2024 GoEdge CDN goedge.cdn@gmail.com. All rights reserved. Official site: https://goedge.cn .

package nodes

import (
	"fmt"
	"github.com/TeaOSLab/EdgeNode/internal/remotelogs"
	"github.com/TeaOSLab/EdgeNode/internal/trackers"
	executils "github.com/TeaOSLab/EdgeNode/internal/utils/exec"
	"github.com/TeaOSLab/EdgeNode/internal/utils/idles"
	"runtime"
	"time"
)

// TrimDisksTask trim ssd disks automatically
type TrimDisksTask struct {
}

// NewTrimDisksTask create new task
func NewTrimDisksTask() *TrimDisksTask {
	return &TrimDisksTask{}
}

// Start the task
func (this *TrimDisksTask) Start() {
	// execute once
	if idles.IsMinHour() {
		err := this.loop()
		if err != nil {
			remotelogs.Warn("TRIM_DISKS", "trim disks failed: "+err.Error())
		}
	}

	var ticker = time.NewTicker(2 * 24 * time.Hour) // every 2 days
	idles.RunTicker(ticker, func() {
		// run the task
		err := this.loop()
		if err != nil {
			remotelogs.Warn("TRIM_DISKS", "trim disks failed: "+err.Error())
		}
	})
}

// run the task once
func (this *TrimDisksTask) loop() error {
	if runtime.GOOS != "linux" {
		return nil
	}

	var nodeConfig = sharedNodeConfig
	if nodeConfig == nil {
		return nil
	}
	if !nodeConfig.AutoTrimDisks {
		return nil
	}

	trimExe, err := executils.LookPath("fstrim")
	if err != nil {
		return fmt.Errorf("'fstrim' command not found: %w", err)
	}

	defer trackers.Begin("TRIM_DISKS").End()

	var cmd = executils.NewCmd(trimExe, "-a").
		WithStderr()
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("'fstrim' execute failed: %s", cmd.Stderr())
	}

	return nil
}
