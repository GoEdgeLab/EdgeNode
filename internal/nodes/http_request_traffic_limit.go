// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package nodes

import (
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs"
)

// 流量限制
func (this *HTTPRequest) doTrafficLimit(status *serverconfigs.TrafficLimitStatus) (blocked bool) {
	if status == nil {
		return false
	}

	// 如果是网站单独设置的流量限制，则检查是否已关闭
	var config = this.ReqServer.TrafficLimit
	if (config == nil || !config.IsOn) && status.PlanId == 0 {
		return false
	}

	// 如果是套餐设置的流量限制，即使套餐变更了（变更套餐或者变更套餐的限制），仍然会提示流量超限

	this.tags = append(this.tags, "trafficLimit")

	var statusCode = 509
	this.writer.statusCode = statusCode
	this.ProcessResponseHeaders(this.writer.Header(), statusCode)

	this.writer.Header().Set("Content-Type", "text/html; charset=utf-8")
	this.writer.WriteHeader(statusCode)

	// check plan traffic limit
	if (config == nil || !config.IsOn) && this.ReqServer.PlanId() > 0 && this.nodeConfig != nil {
		var planConfig = this.nodeConfig.FindPlan(this.ReqServer.PlanId())
		if planConfig != nil && planConfig.TrafficLimit != nil && planConfig.TrafficLimit.IsOn {
			config = planConfig.TrafficLimit
		}
	}

	if config != nil && len(config.NoticePageBody) != 0 {
		_, _ = this.writer.WriteString(this.Format(config.NoticePageBody))
	} else {
		_, _ = this.writer.WriteString(this.Format(serverconfigs.DefaultTrafficLimitNoticePageBody))
	}

	return true
}
