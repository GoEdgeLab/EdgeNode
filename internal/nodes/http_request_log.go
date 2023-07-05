package nodes

import (
	"github.com/TeaOSLab/EdgeCommon/pkg/rpc/pb"
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs"
	"strings"
	"time"
)

const (
	// AccessLogMaxRequestBodySize 访问日志存储的请求内容最大尺寸 TODO 此值应该可以在访问日志页设置
	AccessLogMaxRequestBodySize = 2 * 1024 * 1024
)

// 日志
func (this *HTTPRequest) log() {
	// 检查全局配置
	if this.nodeConfig != nil && this.nodeConfig.GlobalServerConfig != nil && !this.nodeConfig.GlobalServerConfig.HTTPAccessLog.IsOn {
		return
	}

	var ref *serverconfigs.HTTPAccessLogRef
	if !this.forceLog {
		if this.disableLog {
			return
		}

		// 计算请求时间
		this.requestCost = time.Since(this.requestFromTime).Seconds()

		ref = this.web.AccessLogRef
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

		// 是否记录499
		if !ref.EnableClientClosed && this.writer.StatusCode() == 499 {
			return
		}
	}

	var addr = this.RawReq.RemoteAddr
	var index = strings.LastIndex(addr, ":")
	if index > 0 {
		addr = addr[:index]
	}

	var serverGlobalConfig = this.nodeConfig.GlobalServerConfig

	// 请求Cookie
	var cookies = map[string]string{}
	var enableCookies = false
	if serverGlobalConfig == nil || serverGlobalConfig.HTTPAccessLog.EnableCookies {
		enableCookies = true
		if ref == nil || ref.ContainsField(serverconfigs.HTTPAccessLogFieldCookie) {
			for _, cookie := range this.RawReq.Cookies() {
				cookies[cookie.Name] = cookie.Value
			}
		}
	}

	// 请求Header
	var pbReqHeader = map[string]*pb.Strings{}
	if serverGlobalConfig == nil || serverGlobalConfig.HTTPAccessLog.EnableRequestHeaders {
		if ref == nil || ref.ContainsField(serverconfigs.HTTPAccessLogFieldHeader) {
			// 是否只记录通用Header
			var commonHeadersOnly = serverGlobalConfig != nil && serverGlobalConfig.HTTPAccessLog.CommonRequestHeadersOnly

			for k, v := range this.RawReq.Header {
				if commonHeadersOnly && !serverconfigs.IsCommonRequestHeader(k) {
					continue
				}
				if !enableCookies && k == "Cookie" {
					continue
				}
				pbReqHeader[k] = &pb.Strings{Values: v}
			}
		}
	}

	// 响应Header
	var pbResHeader = map[string]*pb.Strings{}
	if serverGlobalConfig == nil || serverGlobalConfig.HTTPAccessLog.EnableResponseHeaders {
		if ref == nil || ref.ContainsField(serverconfigs.HTTPAccessLogFieldSentHeader) {
			for k, v := range this.writer.Header() {
				pbResHeader[k] = &pb.Strings{Values: v}
			}
		}
	}

	// 参数列表
	var queryString = ""
	if ref == nil || ref.ContainsField(serverconfigs.HTTPAccessLogFieldArg) {
		queryString = this.requestQueryString()
	}

	// 浏览器
	var userAgent = ""
	if ref == nil || ref.ContainsField(serverconfigs.HTTPAccessLogFieldUserAgent) || ref.ContainsField(serverconfigs.HTTPAccessLogFieldExtend) {
		userAgent = this.RawReq.UserAgent()
	}

	// 请求来源
	var referer = ""
	if ref == nil || ref.ContainsField(serverconfigs.HTTPAccessLogFieldReferer) {
		referer = this.RawReq.Referer()
	}

	var accessLog = &pb.HTTPAccessLog{
		RequestId:       this.requestId,
		NodeId:          this.nodeConfig.Id,
		ServerId:        this.ReqServer.Id,
		RemoteAddr:      this.requestRemoteAddr(true),
		RawRemoteAddr:   addr,
		RemotePort:      int32(this.requestRemotePort()),
		RemoteUser:      this.requestRemoteUser(),
		RequestURI:      this.rawURI,
		RequestPath:     this.Path(),
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
		Host:            this.ReqHost,
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
		FirewallActions:     this.firewallActions,
		Tags:                this.tags,

		Attrs: this.logAttrs,
	}

	if this.origin != nil {
		accessLog.OriginId = this.origin.Id
		accessLog.OriginAddress = this.originAddr
		accessLog.OriginStatus = this.originStatus
	}

	// 请求Body
	if (ref != nil && ref.ContainsField(serverconfigs.HTTPAccessLogFieldRequestBody)) || this.wafHasRequestBody {
		accessLog.RequestBody = this.requestBodyData

		if len(accessLog.RequestBody) > AccessLogMaxRequestBodySize {
			accessLog.RequestBody = accessLog.RequestBody[:AccessLogMaxRequestBodySize]
		}
	}

	// TODO 记录匹配的 locationId和rewriteId，非必要需求

	sharedHTTPAccessLogQueue.Push(accessLog)
}
