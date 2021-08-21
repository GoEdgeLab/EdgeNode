// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package js

import "net/http"

type RequestInterface interface {
	// JSRequest 请求
	JSRequest() *http.Request

	// JSWriter 响应
	JSWriter() http.ResponseWriter

	// JSStop 中止请求
	JSStop()

	// JSLog 打印日志
	JSLog(msg ...interface{})
}
