package iplibrary

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/TeaOSLab/EdgeCommon/pkg/rpc/pb"
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs/firewallconfigs"
	"os/exec"
	"path/filepath"
)

// 脚本命令动作
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
	cmd := exec.Command(this.config.Path)
	cmd.Env = []string{
		"ACTION=" + action,
		"TYPE=" + item.Type,
		"IP_FROM=" + item.IpFrom,
		"IP_TO=" + item.IpTo,
		"EXPIRED_AT=" + fmt.Sprintf("%d", item.ExpiredAt),
		"LIST_TYPE=" + listType,
	}
	if len(this.config.Cwd) > 0 {
		cmd.Dir = this.config.Cwd
	} else {
		cmd.Dir = filepath.Dir(this.config.Path)
	}
	stderr := bytes.NewBuffer([]byte{})
	cmd.Stderr = stderr
	err := cmd.Run()
	if err != nil {
		return errors.New(err.Error() + ", output: " + string(stderr.Bytes()))
	}
	return nil
}
