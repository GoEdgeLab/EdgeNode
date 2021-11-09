// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package nodes

import (
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs"
	"net/http"
)

// 带宽限制
func (this *HTTPRequest) doPlanExpires() {
	this.tags = append(this.tags, "plan")

	var statusCode = http.StatusNotFound
	this.processResponseHeaders(statusCode)

	this.writer.WriteHeader(statusCode)
	_, _ = this.writer.WriteString(serverconfigs.DefaultPlanExpireNoticePageBody)
}
