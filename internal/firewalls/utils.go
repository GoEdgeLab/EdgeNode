// Copyright 2023 Liuxiangchao iwind.liu@gmail.com. All rights reserved. Official site: https://goedge.cn .

package firewalls

import (
	"time"
)

// DropTemporaryTo 使用本地防火墙临时拦截IP数据包
func DropTemporaryTo(ip string, expiresAt int64) {
	// 如果为0，则表示是长期有效
	if expiresAt <= 0 {
		expiresAt = time.Now().Unix() + 3600
	}

	var timeout = expiresAt - time.Now().Unix()
	if timeout < 1 {
		return
	}
	if timeout > 3600 {
		timeout = 3600
	}

	// 使用本地防火墙延长封禁
	var fw = Firewall()
	if fw != nil && !fw.IsMock() {
		// 这里 int(int64) 转换的前提是限制了 timeout <= 3600，否则将有整型溢出的风险
		_ = fw.DropSourceIP(ip, int(timeout), true)
	}
}
