package iplibrary

import (
	"fmt"
	"github.com/TeaOSLab/EdgeCommon/pkg/rpc/pb"
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs/firewallconfigs"
	executils "github.com/TeaOSLab/EdgeNode/internal/utils/exec"
	"path/filepath"
	"time"
)

// ScriptAction 脚本命令动作
type ScriptAction struct {
	BaseAction

	config *firewallconfigs.FirewallActionScriptConfig
}

func NewScriptAction() *ScriptAction {
	return &ScriptAction{}
}

func (this *ScriptAction) Init(config *firewallconfigs.FirewallActionConfig) error {
	this.config = &firewallconfigs.FirewallActionScriptConfig{}
	err := this.convertParams(config.Params, this.config)
	if err != nil {
		return err
	}

	if len(this.config.Path) == 0 {
		return NewFataError("'path' should not be empty")
	}

	return nil
}

func (this *ScriptAction) AddItem(listType IPListType, item *pb.IPItem) error {
	return this.runAction("addItem", listType, item)
}

func (this *ScriptAction) DeleteItem(listType IPListType, item *pb.IPItem) error {
	return this.runAction("deleteItem", listType, item)
}

func (this *ScriptAction) runAction(action string, listType IPListType, item *pb.IPItem) error {
	// TODO 智能支持 .sh 脚本文件
	var cmd = executils.NewTimeoutCmd(30*time.Second, this.config.Path)
	cmd.WithEnv([]string{
		"ACTION=" + action,
		"TYPE=" + item.Type,
		"IP_FROM=" + item.IpFrom,
		"IP_TO=" + item.IpTo,
		"EXPIRED_AT=" + fmt.Sprintf("%d", item.ExpiredAt),
		"LIST_TYPE=" + listType,
	})
	if len(this.config.Cwd) > 0 {
		cmd.WithDir(this.config.Cwd)
	} else {
		cmd.WithDir(filepath.Dir(this.config.Path))
	}
	cmd.WithStderr()
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("%w, output: %s", err, cmd.Stderr())
	}
	return nil
}
