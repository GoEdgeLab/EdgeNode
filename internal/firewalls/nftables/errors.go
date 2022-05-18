// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved.
//go:build linux
// +build linux

package nftables

import "errors"

var ErrTableNotFound = errors.New("table not found")
var ErrChainNotFound = errors.New("chain not found")
var ErrSetNotFound = errors.New("set not found")
var ErrRuleNotFound = errors.New("rule not found")

func IsNotFound(err error) bool {
	if err == nil {
		return false
	}
	return err == ErrTableNotFound || err == ErrChainNotFound || err == ErrSetNotFound || err == ErrRuleNotFound
}
