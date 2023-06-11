// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package nodes

import (
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs"
	"net/http"
)

// 套餐过期
func (this *HTTPRequest) doPlanExpires() {
	this.tags = append(this.tags, "plan")

	var statusCode = http.StatusNotFound
	this.ProcessResponseHeaders(this.writer.Header(), statusCode)

	this.writer.WriteHeader(statusCode)
	_, _ = this.writer.WriteString(this.Format(serverconfigs.DefaultPlanExpireNoticePageBody))
}
