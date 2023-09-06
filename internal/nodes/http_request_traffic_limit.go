// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package nodes

import (
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs"
)

// 流量限制
func (this *HTTPRequest) doTrafficLimit() {
	this.tags = append(this.tags, "trafficLimit")

	var statusCode = 509
	this.writer.statusCode = statusCode
	this.ProcessResponseHeaders(this.writer.Header(), statusCode)

	this.writer.Header().Set("Content-Type", "text/html; charset=utf-8")
	this.writer.WriteHeader(statusCode)

	var config = this.ReqServer.TrafficLimit
	if config != nil && len(config.NoticePageBody) != 0 {
		_, _ = this.writer.WriteString(this.Format(config.NoticePageBody))
	} else {
		_, _ = this.writer.WriteString(this.Format(serverconfigs.DefaultTrafficLimitNoticePageBody))
	}
}
