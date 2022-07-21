// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved. Official site: https://goedge.cn .

package utils_test

import (
	"github.com/TeaOSLab/EdgeNode/internal/utils"
	"testing"
)

func TestParseAddrHost(t *testing.T) {
	for _, addr := range []string{"a", "example.com", "example.com:1234", "::1", "[::1]", "[::1]:8080"} {
		t.Log(addr + " => " + utils.ParseAddrHost(addr))
	}
}

func TestMergePorts(t *testing.T) {
	for _, ports := range [][]int{
		{},
		{80},
		{80, 83, 85},
		{80, 81, 83, 85, 86, 87, 88, 90},
		{0, 0, 1, 1, 2, 2, 2, 3, 3, 3},
	} {
		t.Log(ports, "=>", utils.MergePorts(ports))
	}
}
