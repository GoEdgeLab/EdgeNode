// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved. Official site: https://goedge.cn .

package clock

import (
	"encoding/binary"
	"fmt"
	"github.com/TeaOSLab/EdgeCommon/pkg/nodeconfigs"
	teaconst "github.com/TeaOSLab/EdgeNode/internal/const"
	"github.com/TeaOSLab/EdgeNode/internal/events"
	"github.com/TeaOSLab/EdgeNode/internal/goman"
	"github.com/TeaOSLab/EdgeNode/internal/remotelogs"
	executils "github.com/TeaOSLab/EdgeNode/internal/utils/exec"
	timeutil "github.com/iwind/TeaGo/utils/time"
	"net"
	"runtime"
	"time"
)

var hasSynced = false
var sharedClockManager = NewClockManager()

func init() {
	if !teaconst.IsMain {
		return
	}

	events.On(events.EventLoaded, func() {
		goman.New(sharedClockManager.Start)
	})
	events.On(events.EventReload, func() {
		if !hasSynced {
			hasSynced = true

			goman.New(func() {
				err := sharedClockManager.Sync()
				if err != nil {
					remotelogs.Warn("CLOCK", "sync clock failed: "+err.Error())
				}
			})
		}
	})
}

type ClockManager struct {
	lastFailAt int64
}

func NewClockManager() *ClockManager {
	return &ClockManager{}
}

// Start 启动
func (this *ClockManager) Start() {
	var ticker = time.NewTicker(1 * time.Hour)
	for range ticker.C {
		err := this.Sync()
		if err != nil {
			var currentTimestamp = time.Now().Unix()

			// 每天只提醒一次错误
			if currentTimestamp-this.lastFailAt > 86400 {
				remotelogs.Warn("CLOCK", "sync clock failed: "+err.Error())
				this.lastFailAt = currentTimestamp
			}
		}
	}
}

// Sync 自动校对时间
func (this *ClockManager) Sync() error {
	if runtime.GOOS != "linux" {
		return nil
	}

	nodeConfig, _ := nodeconfigs.SharedNodeConfig()
	if nodeConfig == nil {
		return nil
	}

	var config = nodeConfig.Clock
	if config == nil || !config.AutoSync {
		return nil
	}

	// check chrony
	if config.CheckChrony {
		chronycExe, err := executils.LookPath("chronyc")
		if err == nil && len(chronycExe) > 0 {
			var chronyCmd = executils.NewTimeoutCmd(3*time.Second, chronycExe, "tracking")
			err = chronyCmd.Run()
			if err == nil {
				return nil
			}
		}
	}

	var server = config.Server
	if len(server) == 0 {
		server = "pool.ntp.org"
	}

	ntpdate, err := executils.LookPath("ntpdate")
	if err != nil {
		// 使用 date 命令设置
		// date --set TIME
		dateExe, err := executils.LookPath("date")
		if err == nil {
			currentTime, err := this.ReadServer(server)
			if err != nil {
				return fmt.Errorf("read server failed: %w", err)
			}

			var delta = time.Now().Unix() - currentTime.Unix()
			if delta > 1 || delta < -1 { // 相差比较大的时候才会同步
				var err = executils.NewTimeoutCmd(3*time.Second, dateExe, "--set", timeutil.Format("Y-m-d H:i:s+P", currentTime)).
					Run()
				if err != nil {
					return err
				}
			}
		}

		return nil
	}
	if len(ntpdate) > 0 {
		return this.syncNtpdate(ntpdate, server)
	}

	return nil
}

func (this *ClockManager) syncNtpdate(ntpdate string, server string) error {
	var cmd = executils.NewTimeoutCmd(30*time.Second, ntpdate, server)
	cmd.WithStderr()
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("%w: %s", err, cmd.Stderr())
	}

	return nil
}

// ReadServer 参考自：https://medium.com/learning-the-go-programming-language/lets-make-an-ntp-client-in-go-287c4b9a969f
func (this *ClockManager) ReadServer(server string) (time.Time, error) {
	conn, err := net.Dial("udp", server+":123")
	if err != nil {
		return time.Time{}, fmt.Errorf("connect to server failed: %w", err)
	}
	defer func() {
		_ = conn.Close()
	}()
	err = conn.SetDeadline(time.Now().Add(5 * time.Second))
	if err != nil {
		return time.Time{}, err
	}

	// configure request settings by specifying the first byte as
	// 00 011 011 (or 0x1B)
	// |  |   +-- client mode (3)
	// |  + ----- version (3)
	// + -------- leap year indicator, 0 no warning

	var req = &NTPPacket{Settings: 0x1B}
	err = binary.Write(conn, binary.BigEndian, req)
	if err != nil {
		return time.Time{}, fmt.Errorf("write request failed: %w", err)
	}

	var resp = &NTPPacket{}
	err = binary.Read(conn, binary.BigEndian, resp)
	if err != nil {
		return time.Time{}, fmt.Errorf("write server response failed: %w", err)
	}

	const ntpEpochOffset = 2208988800

	var secs = float64(resp.TxTimeSec) - ntpEpochOffset
	var nanos = (int64(resp.TxTimeFrac) * 1e9) >> 32
	return time.Unix(int64(secs), nanos), nil
}
