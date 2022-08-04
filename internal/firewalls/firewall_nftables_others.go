// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved.
//go:build !linux
// +build !linux

package firewalls

import (
	"errors"
)

func NewNFTablesFirewall() (*NFTablesFirewall, error) {
	return nil, errors.New("not implemented")
}

type NFTablesFirewall struct {
}

// Name 名称
func (this *NFTablesFirewall) Name() string {
	return "nftables"
}

// IsReady 是否已准备被调用
func (this *NFTablesFirewall) IsReady() bool {
	return false
}

// IsMock 是否为模拟
func (this *NFTablesFirewall) IsMock() bool {
	return true
}

// AllowPort 允许端口
func (this *NFTablesFirewall) AllowPort(port int, protocol string) error {
	return nil
}

// RemovePort 删除端口
func (this *NFTablesFirewall) RemovePort(port int, protocol string) error {
	return nil
}

// AllowSourceIP Allow把IP加入白名单
func (this *NFTablesFirewall) AllowSourceIP(ip string) error {
	return nil
}

// RejectSourceIP 拒绝某个源IP连接
func (this *NFTablesFirewall) RejectSourceIP(ip string, timeoutSeconds int) error {
	return nil
}

// DropSourceIP 丢弃某个源IP数据
func (this *NFTablesFirewall) DropSourceIP(ip string, timeoutSeconds int, async bool) error {
	return nil
}

// RemoveSourceIP 删除某个源IP
func (this *NFTablesFirewall) RemoveSourceIP(ip string) error {
	return nil
}
