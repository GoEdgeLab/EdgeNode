// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved.
//go:build !linux
// +build !linux

package firewalls

import (
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs/ddosconfigs"
)

var SharedDDoSProtectionManager = NewDDoSProtectionManager()

type DDoSProtectionManager struct {
}

func NewDDoSProtectionManager() *DDoSProtectionManager {
	return &DDoSProtectionManager{}
}

func (this *DDoSProtectionManager) Apply(config *ddosconfigs.ProtectionConfig) error {
	return nil
}
