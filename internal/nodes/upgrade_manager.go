// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package nodes

import (
	"crypto/md5"
	"fmt"
	"github.com/TeaOSLab/EdgeCommon/pkg/rpc/pb"
	teaconst "github.com/TeaOSLab/EdgeNode/internal/const"
	"github.com/TeaOSLab/EdgeNode/internal/events"
	"github.com/TeaOSLab/EdgeNode/internal/goman"
	"github.com/TeaOSLab/EdgeNode/internal/remotelogs"
	"github.com/TeaOSLab/EdgeNode/internal/rpc"
	"github.com/TeaOSLab/EdgeNode/internal/utils"
	executils "github.com/TeaOSLab/EdgeNode/internal/utils/exec"
	"github.com/iwind/TeaGo/Tea"
	stringutil "github.com/iwind/TeaGo/utils/string"
	"github.com/iwind/gosock/pkg/gosock"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

var sharedUpgradeManager = NewUpgradeManager()

// UpgradeManager 节点升级管理器
// TODO 需要在集群中设置是否自动更新
type UpgradeManager struct {
	isInstalling bool
	lastFile     string
	exe          string
}

// NewUpgradeManager 获取新对象
func NewUpgradeManager() *UpgradeManager {
	return &UpgradeManager{}
}

// Start 启动升级
func (this *UpgradeManager) Start() {
	// 必须放在文件解压之前读取可执行文件路径，防止解析之后，当前的可执行文件路径发生改变
	exe, err := os.Executable()
	if err != nil {
		remotelogs.Error("UPGRADE_MANAGER", "can not find current executable file name")
		return
	}
	this.exe = exe

	// 测试环境下不更新
	if Tea.IsTesting() {
		return
	}

	if this.isInstalling {
		return
	}
	this.isInstalling = true

	remotelogs.Println("UPGRADE_MANAGER", "upgrading node ...")
	err = this.install()
	if err != nil {
		remotelogs.Error("UPGRADE_MANAGER", "download failed: "+err.Error())

		this.isInstalling = false
		return
	}

	remotelogs.Println("UPGRADE_MANAGER", "upgrade successfully")

	goman.New(func() {
		err = this.restart()
		if err != nil {
			remotelogs.Error("UPGRADE_MANAGER", err.Error())
		}

		this.isInstalling = false
	})
}

// IsInstalling 检查是否正在安装
func (this *UpgradeManager) IsInstalling() bool {
	return this.isInstalling
}

func (this *UpgradeManager) install() error {
	// 检查是否有已下载但未安装成功的
	if len(this.lastFile) > 0 {
		_, err := os.Stat(this.lastFile)
		if err == nil {
			err = this.unzip(this.lastFile)
			if err != nil {
				return err
			}
			this.lastFile = ""
			return nil
		}
	}

	// 创建临时文件
	var dir = Tea.Root + "/tmp"
	_, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			err = os.Mkdir(dir, 0777)
			if err != nil {
				return err
			}
		} else {
			return err
		}
	}

	remotelogs.Println("UPGRADE_MANAGER", "downloading new node ...")

	var path = dir + "/edge-node.tmp"
	fp, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0777)
	if err != nil {
		return err
	}
	isClosed := false
	defer func() {
		if !isClosed {
			_ = fp.Close()
		}
	}()

	client, err := rpc.SharedRPC()
	if err != nil {
		return err
	}

	var offset int64
	var h = md5.New()
	var sum = ""
	var filename = ""
	for {
		resp, err := client.NodeRPC.DownloadNodeInstallationFile(client.Context(), &pb.DownloadNodeInstallationFileRequest{
			Os:          runtime.GOOS,
			Arch:        runtime.GOARCH,
			ChunkOffset: offset,
		})
		if err != nil {
			return err
		}
		if len(resp.Sum) == 0 {
			return nil
		}
		sum = resp.Sum
		filename = resp.Filename
		if stringutil.VersionCompare(resp.Version, teaconst.Version) <= 0 {
			return nil
		}
		if len(resp.ChunkData) == 0 {
			break
		}

		// 写入文件
		_, err = fp.Write(resp.ChunkData)
		if err != nil {
			return err
		}
		_, err = h.Write(resp.ChunkData)
		if err != nil {
			return err
		}

		offset = resp.Offset
	}

	if len(filename) == 0 {
		return nil
	}

	isClosed = true
	err = fp.Close()
	if err != nil {
		return err
	}

	if fmt.Sprintf("%x", h.Sum(nil)) != sum {
		_ = os.Remove(path)
		return nil
	}

	// 改成zip
	zipPath := dir + "/" + filename
	err = os.Rename(path, zipPath)
	if err != nil {
		return err
	}
	this.lastFile = zipPath

	// 解压
	err = this.unzip(zipPath)
	if err != nil {
		return err
	}

	return nil
}

// 解压
func (this *UpgradeManager) unzip(zipPath string) error {
	var isOk = false
	defer func() {
		if isOk {
			// 只有解压并覆盖成功后才会删除
			_ = os.Remove(zipPath)
		}
	}()

	// 解压
	var target = Tea.Root
	if Tea.IsTesting() {
		// 测试环境下只解压在tmp目录
		target = Tea.Root + "/tmp"
	}

	// 先改先前的可执行文件
	err := os.Rename(target+"/bin/"+teaconst.ProcessName, target+"/bin/."+teaconst.ProcessName+".dist")
	var hasBackup = err == nil
	defer func() {
		if !isOk && hasBackup {
			// 失败时还原
			_ = os.Rename(target+"/bin/."+teaconst.ProcessName+".dist", target+"/bin/"+teaconst.ProcessName)
		}
	}()

	var unzip = utils.NewUnzip(zipPath, target, "edge-node/")
	err = unzip.Run()
	if err != nil {
		return err
	}

	isOk = true

	return nil
}

// 重启
func (this *UpgradeManager) restart() error {
	// 关闭当前sock，防止无法重启
	_ = gosock.NewTmpSock(teaconst.ProcessName).Close()

	// 重新启动
	if DaemonIsOn && DaemonPid == os.Getppid() {
		utils.Exit() // TODO 试着更优雅重启
	} else {
		// quit
		events.Notify(events.EventQuit)

		// terminated
		events.Notify(events.EventTerminated)

		// 启动
		var exe = filepath.Dir(this.exe) + "/" + teaconst.ProcessName
		var cmd = executils.NewCmd(exe, "start")
		err := cmd.Start()
		if err != nil {
			return err
		}

		// 退出当前进程
		time.Sleep(1 * time.Second)
		os.Exit(0)
	}
	return nil
}
