package nodes

import "github.com/TeaOSLab/EdgeNode/internal/stats"

// 统计
func (this *HTTPRequest) doStat() {
	if this.Server == nil {
		return
	}
	stats.SharedHTTPRequestStatManager.AddRemoteAddr(this.Server.Id, this.requestRemoteAddr())
	stats.SharedHTTPRequestStatManager.AddUserAgent(this.Server.Id, this.requestHeader("User-Agent"))
}
