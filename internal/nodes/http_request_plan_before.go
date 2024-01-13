// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.
//go:build !plus

package nodes

// 检查套餐
func (this *HTTPRequest) doPlanBefore() (blocked bool) {
	// stub
	return false
}
