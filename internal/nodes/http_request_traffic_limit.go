// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package nodes

import (
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs"
)

// 流量限制
func (this *HTTPRequest) doTrafficLimit() {
	var config = this.ReqServer.TrafficLimit

	this.tags = append(this.tags, "bandwidth")

	var statusCode = 509
	this.processResponseHeaders(statusCode)

	this.writer.WriteHeader(statusCode)
	if len(config.NoticePageBody) != 0 {
		_, _ = this.writer.WriteString(this.Format(config.NoticePageBody))
	} else {
		_, _ = this.writer.WriteString(this.Format(serverconfigs.DefaultTrafficLimitNoticePageBody))
	}
}
