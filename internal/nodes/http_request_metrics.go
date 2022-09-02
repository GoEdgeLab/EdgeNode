// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package nodes

import (
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs"
	"github.com/TeaOSLab/EdgeNode/internal/metrics"
)

// 指标统计 - 响应
// 只需要在结束时调用指标进行统计
func (this *HTTPRequest) doMetricsResponse() {
	metrics.SharedManager.Add(this)
}

func (this *HTTPRequest) MetricKey(key string) string {
	return this.Format(key)
}

func (this *HTTPRequest) MetricValue(value string) (result int64, ok bool) {
	// TODO 需要忽略健康检查的请求，但是同时也要防止攻击者模拟健康检查
	switch value {
	case "${countRequest}":
		return 1, true
	case "${countTrafficOut}":
		// 这里不包括Header长度
		return this.writer.SentBodyBytes(), true
	case "${countTrafficIn}":
		var hl int64 = 0 // header length
		for k, values := range this.RawReq.Header {
			for _, v := range values {
				hl += int64(len(k) + len(v) + 2 /** k: v  **/)
			}
		}
		return this.RawReq.ContentLength + hl, true
	case "${countConnection}":
		return 1, true
	}
	return 0, false
}

func (this *HTTPRequest) MetricServerId() int64 {
	return this.ReqServer.Id
}

func (this *HTTPRequest) MetricCategory() string {
	return serverconfigs.MetricItemCategoryHTTP
}
