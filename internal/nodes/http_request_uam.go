// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved.
//go:build !plus
// +build !plus

package nodes

// UAM
func (this *HTTPRequest) doUAM() (block bool) {
	return false
}
