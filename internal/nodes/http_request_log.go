package nodes

import (
	"github.com/TeaOSLab/EdgeCommon/pkg/rpc/pb"
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

var requestId int64 = 1_0000_0000_0000_0000

// 日志
func (this *HTTPRequest) log() {
	if this.disableLog {
		return
	}

	// 计算请求时间
	this.requestCost = time.Since(this.requestFromTime).Seconds()

	ref := this.web.AccessLogRef
	if ref == nil {
		ref = serverconfigs.DefaultHTTPAccessLogRef
	}
	if !ref.IsOn {
		return
	}

	if !ref.Match(this.writer.StatusCode()) {
		return
	}

	if ref.FirewallOnly && this.firewallPolicyId == 0 {
		return
	}

	addr := this.RawReq.RemoteAddr
	index := strings.LastIndex(addr, ":")
	if index > 0 {
		addr = addr[:index]
	}

	// 请求Cookie
	cookies := map[string]string{}
	if ref.ContainsField(serverconfigs.HTTPAccessLogFieldCookie) {
		for _, cookie := range this.RawReq.Cookies() {
			cookies[cookie.Name] = cookie.Value
		}
	}

	// 请求Header
	pbReqHeader := map[string]*pb.Strings{}
	if ref.ContainsField(serverconfigs.HTTPAccessLogFieldHeader) {
		for k, v := range this.RawReq.Header {
			pbReqHeader[k] = &pb.Strings{Values: v}
		}
	}

	// 响应Header
	pbResHeader := map[string]*pb.Strings{}
	if ref.ContainsField(serverconfigs.HTTPAccessLogFieldSentHeader) {
		for k, v := range this.writer.Header() {
			pbResHeader[k] = &pb.Strings{Values: v}
		}
	}

	// 参数列表
	queryString := ""
	if ref.ContainsField(serverconfigs.HTTPAccessLogFieldArg) {
		queryString = this.requestQueryString()
	}

	// 浏览器
	userAgent := ""
	if ref.ContainsField(serverconfigs.HTTPAccessLogFieldUserAgent) || ref.ContainsField(serverconfigs.HTTPAccessLogFieldExtend) {
		userAgent = this.RawReq.UserAgent()
	}

	// 请求来源
	referer := ""
	if ref.ContainsField(serverconfigs.HTTPAccessLogFieldReferer) {
		referer = this.RawReq.Referer()
	}

	accessLog := &pb.HTTPAccessLog{
		RequestId:       strconv.FormatInt(this.requestFromTime.UnixNano(), 10) + strconv.FormatInt(atomic.AddInt64(&requestId, 1), 10) + sharedNodeConfig.PaddedId(),
		NodeId:          sharedNodeConfig.Id,
		ServerId:        this.Server.Id,
		RemoteAddr:      this.requestRemoteAddr(),
		RawRemoteAddr:   addr,
		RemotePort:      int32(this.requestRemotePort()),
		RemoteUser:      this.requestRemoteUser(),
		RequestURI:      this.rawURI,
		RequestPath:     this.requestPath(),
		RequestLength:   this.requestLength(),
		RequestTime:     this.requestCost,
		RequestMethod:   this.RawReq.Method,
		RequestFilename: this.requestFilename(),
		Scheme:          this.requestScheme(),
		Proto:           this.RawReq.Proto,
		BytesSent:       this.writer.SentBodyBytes(), // TODO 加上Header Size
		BodyBytesSent:   this.writer.SentBodyBytes(),
		Status:          int32(this.writer.StatusCode()),
		StatusMessage:   "",
		TimeISO8601:     this.requestFromTime.Format("2006-01-02T15:04:05.000Z07:00"),
		TimeLocal:       this.requestFromTime.Format("2/Jan/2006:15:04:05 -0700"),
		Msec:            float64(this.requestFromTime.Unix()) + float64(this.requestFromTime.Nanosecond())/1000000000,
		Timestamp:       this.requestFromTime.Unix(),
		Host:            this.Host,
		Referer:         referer,
		UserAgent:       userAgent,
		Request:         this.requestString(),
		ContentType:     this.writer.Header().Get("Content-Type"),
		Cookie:          cookies,
		Args:            queryString,
		QueryString:     queryString,
		Header:          pbReqHeader,
		ServerName:      this.ServerName,
		ServerPort:      int32(this.requestServerPort()),
		ServerProtocol:  this.RawReq.Proto,
		SentHeader:      pbResHeader,
		Errors:          this.errors,
		Hostname:        HOSTNAME,

		FirewallPolicyId:    this.firewallPolicyId,
		FirewallRuleGroupId: this.firewallRuleGroupId,
		FirewallRuleSetId:   this.firewallRuleSetId,
		FirewallRuleId:      this.firewallRuleId,

		Attrs: this.logAttrs,
	}

	if this.origin != nil {
		accessLog.OriginId = this.origin.Id
		accessLog.OriginAddress = this.originAddr
	}

	// TODO 记录匹配的 locationId和rewriteId

	sharedHTTPAccessLogQueue.Push(accessLog)
}
