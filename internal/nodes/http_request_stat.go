package nodes

import (
	"github.com/TeaOSLab/EdgeNode/internal/stats"
)

// 统计
func (this *HTTPRequest) doStat() {
	if this.ReqServer == nil || this.web == nil || this.web.StatRef == nil {
		return
	}

	// 内置的统计
	stats.SharedHTTPRequestStatManager.AddRemoteAddr(this.ReqServer.Id, this.requestRemoteAddr(true), this.writer.SentBodyBytes(), this.isAttack)
	stats.SharedHTTPRequestStatManager.AddUserAgent(this.ReqServer.Id, this.requestHeader("User-Agent"), this.remoteAddr)
}
