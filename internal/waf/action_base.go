// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package waf

import (
	"net/http"
)

type BaseAction struct {
}

// CloseConn 关闭连接
func (this *BaseAction) CloseConn(writer http.ResponseWriter) error {
	// 断开连接
	hijack, ok := writer.(http.Hijacker)
	if ok {
		conn, _, err := hijack.Hijack()
		if err == nil && conn != nil {
			return conn.Close()
		}
	}
	return nil
}
