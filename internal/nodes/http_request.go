package nodes

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/TeaOSLab/EdgeCommon/pkg/configutils"
	iplib "github.com/TeaOSLab/EdgeCommon/pkg/iplibrary"
	"github.com/TeaOSLab/EdgeCommon/pkg/nodeconfigs"
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs"
	teaconst "github.com/TeaOSLab/EdgeNode/internal/const"
	"github.com/TeaOSLab/EdgeNode/internal/metrics"
	"github.com/TeaOSLab/EdgeNode/internal/stats"
	"github.com/TeaOSLab/EdgeNode/internal/utils"
	"github.com/iwind/TeaGo/lists"
	"github.com/iwind/TeaGo/maps"
	"github.com/iwind/TeaGo/types"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// 环境变量
var HOSTNAME, _ = os.Hostname()

// errors
var errWritingToClient = errors.New("writing to client error")

// HTTPRequest HTTP请求
type HTTPRequest struct {
	requestId string

	// 外部参数
	RawReq        *http.Request
	RawWriter     http.ResponseWriter
	ReqServer     *serverconfigs.ServerConfig
	ReqHost       string // 请求的Host
	ServerName    string // 实际匹配到的Host
	ServerAddr    string // 实际启动的服务器监听地址
	IsHTTP        bool
	IsHTTPS       bool
	IsHTTP3       bool
	isHealthCheck bool

	// 共享参数
	nodeConfig *nodeconfigs.NodeConfig

	// ln request
	isLnRequest  bool
	lnRemoteAddr string

	// 内部参数
	isSubRequest         bool
	writer               *HTTPWriter
	web                  *serverconfigs.HTTPWebConfig      // Web配置，重要提示：由于引用了别的共享的配置，所以操作中只能读取不要修改
	reverseProxyRef      *serverconfigs.ReverseProxyRef    // 反向代理引用
	reverseProxy         *serverconfigs.ReverseProxyConfig // 反向代理配置，重要提示：由于引用了别的共享的配置，所以操作中只能读取不要修改
	rawURI               string                            // 原始的URI
	uri                  string                            // 经过rewrite等运算之后的URI
	varMapping           map[string]string                 // 变量集合
	requestFromTime      time.Time                         // 请求开始时间
	requestCost          float64                           // 请求耗时
	filePath             string                            // 请求的文件名，仅在读取Root目录下的内容时不为空
	origin               *serverconfigs.OriginConfig       // 源站
	originAddr           string                            // 源站实际地址
	originStatus         int32                             // 源站响应代码
	errors               []string                          // 错误信息
	rewriteRule          *serverconfigs.HTTPRewriteRule    // 匹配到的重写规则
	rewriteReplace       string                            // 重写规则的目标
	rewriteIsExternalURL bool                              // 重写目标是否为外部URL
	remoteAddr           string                            // 计算后的RemoteAddr

	cacheRef         *serverconfigs.HTTPCacheRef // 缓存设置
	cacheKey         string                      // 缓存使用的Key
	isCached         bool                        // 是否已经被缓存
	cacheCanTryStale bool                        // 是否可以尝试使用Stale缓存

	isAttack        bool   // 是否是攻击请求
	requestBodyData []byte // 读取的Body内容

	// WAF相关
	firewallPolicyId    int64
	firewallRuleGroupId int64
	firewallRuleSetId   int64
	firewallRuleId      int64
	firewallActions     []string
	wafHasRequestBody   bool

	tags []string

	logAttrs map[string]string

	disableLog bool // 是否在当前请求中关闭Log
	forceLog   bool // 是否强制记录日志

	// script相关操作
	isDone bool
}

// 初始化
func (this *HTTPRequest) init() {
	this.writer = NewHTTPWriter(this, this.RawWriter)
	this.web = &serverconfigs.HTTPWebConfig{
		IsOn: true,
	}

	// this.uri = this.RawReq.URL.RequestURI()
	// 之所以不使用RequestURI()，是不想让URL中的Path被Encode
	var urlPath = this.RawReq.URL.Path
	if this.ReqServer.Web != nil && this.ReqServer.Web.MergeSlashes {
		urlPath = utils.CleanPath(urlPath)
		this.web.MergeSlashes = true
	}
	if len(this.RawReq.URL.RawQuery) > 0 {
		this.uri = urlPath + "?" + this.RawReq.URL.RawQuery
	} else {
		this.uri = urlPath
	}

	this.rawURI = this.uri
	this.varMapping = map[string]string{
		// 缓存相关初始化
		"cache.status":      "BYPASS",
		"cache.age":         "0",
		"cache.key":         "",
		"cache.policy.name": "",
		"cache.policy.id":   "0",
		"cache.policy.type": "",
	}
	this.logAttrs = map[string]string{}
	this.requestFromTime = time.Now()
	this.requestId = httpRequestNextId()
}

// Do 执行请求
func (this *HTTPRequest) Do() {
	// 初始化
	this.init()

	// 当前服务的反向代理配置
	if this.ReqServer.ReverseProxyRef != nil && this.ReqServer.ReverseProxy != nil {
		this.reverseProxyRef = this.ReqServer.ReverseProxyRef
		this.reverseProxy = this.ReqServer.ReverseProxy
	}

	// Web配置
	err := this.configureWeb(this.ReqServer.Web, true, 0)
	if err != nil {
		this.write50x(err, http.StatusInternalServerError, "Failed to configure the server", "配置服务失败", false)
		this.doEnd()
		return
	}

	// 是否为低级别节点
	this.isLnRequest = this.checkLnRequest()

	// 回调事件
	this.onInit()
	if this.writer.isFinished {
		this.doEnd()
		return
	}

	// 处理健康检查
	var healthCheckKey = this.RawReq.Header.Get(serverconfigs.HealthCheckHeaderName)
	if len(healthCheckKey) > 0 {
		if this.doHealthCheck(healthCheckKey, &this.isHealthCheck) {
			this.doEnd()
			return
		}
	}

	if !this.isLnRequest {
		// 特殊URL处理
		if len(this.rawURI) > 1 && this.rawURI[1] == '.' {
			// ACME
			// TODO 需要配置是否启用ACME检测
			if strings.HasPrefix(this.rawURI, "/.well-known/acme-challenge/") {
				if this.doACME() {
					this.doEnd()
					return
				}
			}
		}

		// 套餐
		if this.ReqServer.UserPlan != nil && !this.ReqServer.UserPlan.IsAvailable() {
			this.doPlanExpires()
			this.doEnd()
			return
		}

		// 流量限制
		if this.ReqServer.TrafficLimitStatus != nil && this.ReqServer.TrafficLimitStatus.IsValid() {
			this.doTrafficLimit()
			this.doEnd()
			return
		}

		// WAF
		if this.web.FirewallRef != nil && this.web.FirewallRef.IsOn {
			if this.doWAFRequest() {
				this.doEnd()
				return
			}
		}

		// UAM
		if !this.isHealthCheck {
			if this.web.UAM != nil {
				if this.web.UAM.IsOn {
					if this.doUAM() {
						this.doEnd()
						return
					}
				}
			} else if this.ReqServer.UAM != nil && this.ReqServer.UAM.IsOn {
				this.web.UAM = this.ReqServer.UAM
				if this.doUAM() {
					this.doEnd()
					return
				}
			}
		}

		// CC
		if !this.isHealthCheck {
			if this.web.CC != nil {
				if this.web.CC.IsOn {
					if this.doCC() {
						this.doEnd()
						return
					}
				}
			}
		}

		// 防盗链
		if !this.isSubRequest && this.web.Referers != nil && this.web.Referers.IsOn {
			if this.doCheckReferers() {
				this.doEnd()
				return
			}
		}

		// UA名单
		if !this.isSubRequest && this.web.UserAgent != nil && this.web.UserAgent.IsOn {
			if this.doCheckUserAgent() {
				this.doEnd()
				return
			}
		}

		// 访问控制
		if !this.isSubRequest && this.web.Auth != nil && this.web.Auth.IsOn {
			if this.doAuth() {
				this.doEnd()
				return
			}
		}

		// 自动跳转到HTTPS
		if this.IsHTTP && this.web.RedirectToHttps != nil && this.web.RedirectToHttps.IsOn {
			if this.doRedirectToHTTPS(this.web.RedirectToHttps) {
				this.doEnd()
				return
			}
		}

		// Compression
		if this.web.Compression != nil && this.web.Compression.IsOn && this.web.Compression.Level > 0 {
			this.writer.SetCompression(this.web.Compression)
		}
	}

	// 开始调用
	this.doBegin()

	// 关闭写入
	this.writer.Close()

	// 结束调用
	this.doEnd()
}

// 开始调用
func (this *HTTPRequest) doBegin() {
	// 是否找不到域名匹配
	if this.ReqServer.Id == 0 {
		this.doMismatch()
		return
	}

	if !this.isLnRequest {
		// 处理request limit
		if this.web.RequestLimit != nil &&
			this.web.RequestLimit.IsOn {
			if this.doRequestLimit() {
				return
			}
		}

		// 处理requestBody
		if this.RawReq.ContentLength > 0 &&
			this.web.AccessLogRef != nil &&
			this.web.AccessLogRef.IsOn &&
			this.web.AccessLogRef.ContainsField(serverconfigs.HTTPAccessLogFieldRequestBody) {
			var err error
			this.requestBodyData, err = io.ReadAll(io.LimitReader(this.RawReq.Body, AccessLogMaxRequestBodySize))
			if err != nil {
				this.write50x(err, http.StatusBadGateway, "Failed to read request body for access log", "为访问日志读取请求Body失败", false)
				return
			}
			this.RawReq.Body = io.NopCloser(io.MultiReader(bytes.NewBuffer(this.requestBodyData), this.RawReq.Body))
		}

		// 跳转
		if len(this.web.HostRedirects) > 0 {
			if this.doHostRedirect() {
				return
			}
		}

		// 临时关闭页面
		if this.web.Shutdown != nil && this.web.Shutdown.IsOn {
			this.doShutdown()
			return
		}
	}

	// 缓存
	if this.web.Cache != nil && this.web.Cache.IsOn {
		if this.doCacheRead(false) {
			return
		}
	}

	if !this.isLnRequest {
		// 重写规则
		if this.rewriteRule != nil {
			if this.doRewrite() {
				return
			}
		}

		// Fastcgi
		if this.web.FastcgiRef != nil && this.web.FastcgiRef.IsOn && len(this.web.FastcgiList) > 0 {
			if this.doFastcgi() {
				return
			}
		}

		// root
		if this.web.Root != nil && this.web.Root.IsOn {
			// 如果处理成功，则终止请求的处理
			if this.doRoot() {
				return
			}

			// 如果明确设置了终止，则也会自动终止
			if this.web.Root.IsBreak {
				return
			}
		}
	}

	// Reverse Proxy
	if this.reverseProxyRef != nil && this.reverseProxyRef.IsOn && this.reverseProxy != nil && this.reverseProxy.IsOn {
		this.doReverseProxy()
		return
	}

	// 返回404页面
	this.write404()
}

// 结束调用
func (this *HTTPRequest) doEnd() {
	// 记录日志
	this.log()

	// 流量统计
	// TODO 增加是否开启开关
	if this.ReqServer != nil && this.ReqServer.Id > 0 && !this.isHealthCheck /** 健康检查时不统计 **/ {
		var totalBytes int64 = 0

		var requestConn = this.RawReq.Context().Value(HTTPConnContextKey)
		if requestConn != nil {
			requestClientConn, ok := requestConn.(ClientConnInterface)
			if ok {
				// 这里读取的其实是上一个请求消耗的流量，不是当前请求消耗的流量，只不过单个请求的流量统计不需要特别精确，整体趋于一致即可
				totalBytes = requestClientConn.LastRequestBytes()
			}
		}

		if totalBytes == 0 {
			totalBytes = this.writer.SentBodyBytes() + this.writer.SentHeaderBytes()
		}

		var countCached int64 = 0
		var cachedBytes int64 = 0

		var countAttacks int64 = 0
		var attackBytes int64 = 0

		if this.isCached {
			countCached = 1
			cachedBytes = totalBytes
		}
		if this.isAttack {
			countAttacks = 1
			attackBytes = this.CalculateSize()
			if attackBytes < totalBytes {
				attackBytes = totalBytes
			}
		}

		stats.SharedTrafficStatManager.Add(this.ReqServer.UserId, this.ReqServer.Id, this.ReqHost, totalBytes, cachedBytes, 1, countCached, countAttacks, attackBytes, this.ReqServer.ShouldCheckTrafficLimit(), this.ReqServer.PlanId())

		// 指标
		if metrics.SharedManager.HasHTTPMetrics() {
			this.doMetricsResponse()
		}

		// 统计
		if this.web.StatRef != nil && this.web.StatRef.IsOn {
			// 放到最后执行
			this.doStat()
		}
	}
}

// RawURI 原始的请求URI
func (this *HTTPRequest) RawURI() string {
	return this.rawURI
}

// 配置
func (this *HTTPRequest) configureWeb(web *serverconfigs.HTTPWebConfig, isTop bool, redirects int) error {
	if web == nil || !web.IsOn {
		return nil
	}

	// 防止跳转次数过多
	if redirects > 8 {
		return errors.New("too many redirects")
	}
	redirects++

	// uri
	rawPath := ""
	rawQuery := ""
	qIndex := strings.Index(this.uri, "?") // question mark index
	if qIndex > -1 {
		rawPath = this.uri[:qIndex]
		rawQuery = this.uri[qIndex+1:]
	} else {
		rawPath = this.uri
	}

	// redirect
	if web.RedirectToHttps != nil && (web.RedirectToHttps.IsPrior || isTop) {
		this.web.RedirectToHttps = web.RedirectToHttps
	}

	// pages
	if len(web.Pages) > 0 {
		this.web.Pages = web.Pages
	}

	// shutdown
	if web.Shutdown != nil && (web.Shutdown.IsPrior || isTop) {
		this.web.Shutdown = web.Shutdown
	}

	// headers
	if web.RequestHeaderPolicyRef != nil && (web.RequestHeaderPolicyRef.IsPrior || isTop) && web.RequestHeaderPolicy != nil {
		// TODO 现在是只能选一个有效的设置，未来可以选择是否合并多级别的设置
		this.web.RequestHeaderPolicy = web.RequestHeaderPolicy
	}
	if web.ResponseHeaderPolicyRef != nil && (web.ResponseHeaderPolicyRef.IsPrior || isTop) && web.ResponseHeaderPolicy != nil {
		// TODO 现在是只能选一个有效的设置，未来可以选择是否合并多级别的设置
		this.web.ResponseHeaderPolicy = web.ResponseHeaderPolicy
	}

	// root
	if web.Root != nil && (web.Root.IsPrior || isTop) {
		this.web.Root = web.Root
	}

	// remote addr
	if web.RemoteAddr != nil && (web.RemoteAddr.IsPrior || isTop) && web.RemoteAddr.IsOn {
		this.web.RemoteAddr = web.RemoteAddr

		// check if from proxy
		if len(this.web.RemoteAddr.Value) > 0 && this.web.RemoteAddr.Value != "${rawRemoteAddr}" {
			var requestConn = this.RawReq.Context().Value(HTTPConnContextKey)
			if requestConn != nil {
				requestClientConn, ok := requestConn.(ClientConnInterface)
				if ok {
					requestClientConn.SetIsPersistent(true)
				}
			}
		}
	}

	// charset
	if web.Charset != nil && (web.Charset.IsPrior || isTop) {
		this.web.Charset = web.Charset
	}

	// websocket
	if web.WebsocketRef != nil && (web.WebsocketRef.IsPrior || isTop) {
		this.web.WebsocketRef = web.WebsocketRef
		this.web.Websocket = web.Websocket
	}

	// compression
	if web.Compression != nil && (web.Compression.IsPrior || isTop) {
		this.web.Compression = web.Compression
	}

	// optimizer
	if web.Optimization != nil && (web.Optimization.IsPrior || (isTop && web.Optimization.IsOn())) {
		this.web.Optimization = web.Optimization
	}

	// webp
	if web.WebP != nil && (web.WebP.IsPrior || isTop) {
		this.web.WebP = web.WebP
	}

	// cache
	if web.Cache != nil && (web.Cache.IsPrior || isTop) {
		this.web.Cache = web.Cache
	}

	// waf
	if web.FirewallRef != nil && (web.FirewallRef.IsPrior || isTop) {
		this.web.FirewallRef = web.FirewallRef
		if web.FirewallPolicy != nil {
			this.web.FirewallPolicy = web.FirewallPolicy
		}
	}

	// access log
	if web.AccessLogRef != nil && (web.AccessLogRef.IsPrior || isTop) {
		this.web.AccessLogRef = web.AccessLogRef
	}

	// host redirects
	if len(web.HostRedirects) > 0 {
		this.web.HostRedirects = web.HostRedirects
	}

	// stat
	if web.StatRef != nil && (web.StatRef.IsPrior || isTop) {
		this.web.StatRef = web.StatRef
	}

	// fastcgi
	if web.FastcgiRef != nil && (web.FastcgiRef.IsPrior || isTop) {
		this.web.FastcgiRef = web.FastcgiRef
		this.web.FastcgiList = web.FastcgiList
	}

	// auth
	if web.Auth != nil && (web.Auth.IsPrior || isTop) {
		this.web.Auth = web.Auth
	}

	// referers
	if web.Referers != nil && (web.Referers.IsPrior || isTop) {
		this.web.Referers = web.Referers
	}

	// user agent
	if web.UserAgent != nil && (web.UserAgent.IsPrior || isTop) {
		this.web.UserAgent = web.UserAgent
	}

	// request limit
	if web.RequestLimit != nil && (web.RequestLimit.IsPrior || isTop) {
		this.web.RequestLimit = web.RequestLimit
	}

	// request scripts
	if web.RequestScripts != nil {
		if this.web.RequestScripts == nil {
			this.web.RequestScripts = &serverconfigs.HTTPRequestScriptsConfig{
				InitGroup:    web.RequestScripts.InitGroup,
				RequestGroup: web.RequestScripts.RequestGroup,
			} // 不要直接赋值，需要复制，防止在运行时被修改
		} else {
			if web.RequestScripts.InitGroup != nil && (web.RequestScripts.InitGroup.IsPrior || isTop) {
				if this.web.RequestScripts == nil {
					this.web.RequestScripts = &serverconfigs.HTTPRequestScriptsConfig{}
				}
				this.web.RequestScripts.InitGroup = web.RequestScripts.InitGroup
			}
			if web.RequestScripts.RequestGroup != nil && (web.RequestScripts.RequestGroup.IsPrior || isTop) {
				if this.web.RequestScripts == nil {
					this.web.RequestScripts = &serverconfigs.HTTPRequestScriptsConfig{}
				}
				this.web.RequestScripts.RequestGroup = web.RequestScripts.RequestGroup
			}
		}
	}

	// UAM
	if web.UAM != nil && (web.UAM.IsPrior || isTop) {
		this.web.UAM = web.UAM
	}

	// CC
	if web.CC != nil && (web.CC.IsPrior || isTop) {
		this.web.CC = web.CC
	}

	// 重写规则
	if len(web.RewriteRefs) > 0 {
		for index, ref := range web.RewriteRefs {
			if !ref.IsOn {
				continue
			}
			rewriteRule := web.RewriteRules[index]
			if !rewriteRule.IsOn {
				continue
			}
			if replace, varMapping, isMatched := rewriteRule.MatchRequest(rawPath, this.Format); isMatched {
				this.addVarMapping(varMapping)
				this.rewriteRule = rewriteRule

				if rewriteRule.WithQuery {
					queryIndex := strings.Index(replace, "?")
					if queryIndex > -1 {
						if len(rawQuery) > 0 {
							replace = replace[:queryIndex] + "?" + rawQuery + "&" + replace[queryIndex+1:]
						}
					} else {
						if len(rawQuery) > 0 {
							replace += "?" + rawQuery
						}
					}
				}

				this.rewriteReplace = replace

				// 如果是外部URL直接返回
				if rewriteRule.IsExternalURL(replace) {
					this.rewriteIsExternalURL = true
					return nil
				}

				// 如果是内部URL继续解析
				if replace == this.uri {
					// URL不变，则停止解析，防止无限循环跳转
					return nil
				}
				this.uri = replace

				// 终止解析的几个个条件：
				//    isBreak = true
				//    mode = redirect
				//    replace = external url
				//    replace = uri
				if rewriteRule.IsBreak || rewriteRule.Mode == serverconfigs.HTTPRewriteModeRedirect {
					return nil
				}

				return this.configureWeb(web, isTop, redirects+1)
			}
		}
	}

	// locations
	if len(web.LocationRefs) > 0 {
		var resultLocation *serverconfigs.HTTPLocationConfig
		for index, ref := range web.LocationRefs {
			if !ref.IsOn {
				continue
			}
			location := web.Locations[index]
			if !location.IsOn {
				continue
			}
			if varMapping, isMatched := location.Match(rawPath, this.Format); isMatched {
				// 检查专属域名
				if len(location.Domains) > 0 && !configutils.MatchDomains(location.Domains, this.ReqHost) {
					continue
				}

				if len(varMapping) > 0 {
					this.addVarMapping(varMapping)
				}
				resultLocation = location

				if location.IsBreak {
					break
				}
			}
		}
		if resultLocation != nil {
			// reset rewrite rule
			this.rewriteRule = nil

			// Reverse Proxy
			if resultLocation.ReverseProxyRef != nil && resultLocation.ReverseProxyRef.IsPrior {
				this.reverseProxyRef = resultLocation.ReverseProxyRef
				this.reverseProxy = resultLocation.ReverseProxy
			}

			// Web
			if resultLocation.Web != nil {
				err := this.configureWeb(resultLocation.Web, false, redirects+1)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// Format 利用请求参数格式化字符串
func (this *HTTPRequest) Format(source string) string {
	if len(source) == 0 {
		return ""
	}

	var hasVarMapping = len(this.varMapping) > 0

	return configutils.ParseVariables(source, func(varName string) string {
		// 自定义变量
		if hasVarMapping {
			value, found := this.varMapping[varName]
			if found {
				return value
			}
		}

		// 请求变量
		switch varName {
		case "edgeVersion":
			return teaconst.Version
		case "remoteAddr":
			return this.requestRemoteAddr(true)
		case "remoteAddrValue":
			return this.requestRemoteAddr(false)
		case "rawRemoteAddr":
			var addr = this.RawReq.RemoteAddr
			host, _, err := net.SplitHostPort(addr)
			if err == nil {
				addr = host
			}
			return addr
		case "remotePort":
			return strconv.Itoa(this.requestRemotePort())
		case "remoteUser":
			return this.requestRemoteUser()
		case "requestId":
			return this.requestId
		case "requestURI", "requestUri":
			return this.rawURI
		case "requestURL":
			var scheme = "http"
			if this.IsHTTPS {
				scheme = "https"
			}
			return scheme + "://" + this.ReqHost + this.rawURI
		case "requestPath":
			return this.Path()
		case "requestPathExtension":
			return filepath.Ext(this.Path())
		case "requestPathLowerExtension":
			return strings.ToLower(filepath.Ext(this.Path()))
		case "requestLength":
			return strconv.FormatInt(this.requestLength(), 10)
		case "requestTime":
			return fmt.Sprintf("%.6f", this.requestCost)
		case "requestMethod":
			return this.RawReq.Method
		case "requestFilename":
			filename := this.requestFilename()
			if len(filename) > 0 {
				return filename
			}

			if this.web.Root != nil && this.web.Root.IsOn {
				return filepath.Clean(this.web.Root.Dir + this.Path())
			}

			return ""
		case "scheme":
			if this.IsHTTP {
				return "http"
			} else {
				return "https"
			}
		case "serverProtocol", "proto":
			return this.RawReq.Proto
		case "bytesSent":
			return strconv.FormatInt(this.writer.SentBodyBytes(), 10) // TODO 加上Header长度
		case "bodyBytesSent":
			return strconv.FormatInt(this.writer.SentBodyBytes(), 10)
		case "status":
			return strconv.Itoa(this.writer.StatusCode())
		case "statusMessage":
			return http.StatusText(this.writer.StatusCode())
		case "timeISO8601":
			return this.requestFromTime.Format("2006-01-02T15:04:05.000Z07:00")
		case "timeLocal":
			return this.requestFromTime.Format("2/Jan/2006:15:04:05 -0700")
		case "msec":
			return fmt.Sprintf("%.6f", float64(this.requestFromTime.Unix())+float64(this.requestFromTime.Nanosecond())/1000000000)
		case "timestamp":
			return strconv.FormatInt(this.requestFromTime.Unix(), 10)
		case "host":
			return this.ReqHost
		case "cname":
			return this.ReqServer.CNameDomain
		case "referer":
			return this.RawReq.Referer()
		case "referer.host":
			u, err := url.Parse(this.RawReq.Referer())
			if err == nil {
				return u.Host
			}
			return ""
		case "userAgent":
			return this.RawReq.UserAgent()
		case "contentType":
			return this.requestContentType()
		case "request":
			return this.requestString()
		case "cookies":
			return this.requestCookiesString()
		case "isArgs":
			if strings.Contains(this.uri, "?") {
				return "?"
			}
			return ""
		case "args", "queryString":
			return this.requestQueryString()
		case "headers":
			return this.requestHeadersString()
		case "serverName":
			return this.ServerName
		case "serverAddr":
			var nodeConfig = this.nodeConfig
			if nodeConfig != nil && nodeConfig.GlobalServerConfig != nil && nodeConfig.GlobalServerConfig.HTTPAll.EnableServerAddrVariable {
				if len(this.requestRemoteAddrs()) > 1 {
					return "" // hidden for security
				}
				var requestConn = this.RawReq.Context().Value(HTTPConnContextKey)
				if requestConn != nil {
					conn, ok := requestConn.(net.Conn)
					if ok {
						host, _, _ := net.SplitHostPort(conn.LocalAddr().String())
						if len(host) > 0 {
							return host
						}
					}
				}
			}
			return ""
		case "serverPort":
			return strconv.Itoa(this.requestServerPort())
		case "hostname":
			return HOSTNAME
		case "documentRoot":
			if this.web.Root != nil {
				return this.web.Root.Dir
			}
			return ""
		}

		dotIndex := strings.Index(varName, ".")
		if dotIndex < 0 {
			return "${" + varName + "}"
		}
		prefix := varName[:dotIndex]
		suffix := varName[dotIndex+1:]

		// cookie.
		if prefix == "cookie" {
			return this.requestCookie(suffix)
		}

		// arg.
		if prefix == "arg" {
			return this.requestQueryParam(suffix)
		}

		// header.
		if prefix == "header" || prefix == "http" {
			return this.requestHeader(suffix)
		}

		// response.
		if prefix == "response" {
			switch suffix {
			case "contentType":
				return this.writer.Header().Get("Content-Type")
			}

			// response.xxx.xxx
			dotIndex = strings.Index(suffix, ".")
			if dotIndex < 0 {
				return "${" + varName + "}"
			}
			switch suffix[:dotIndex] {
			case "header":
				var headers = this.writer.Header()
				var headerKey = suffix[dotIndex+1:]
				v, found := headers[headerKey]
				if found {
					if len(v) == 0 {
						return ""
					}
					return v[0]
				}
				var canonicalHeaderKey = http.CanonicalHeaderKey(headerKey)
				if canonicalHeaderKey != headerKey {
					v = headers[canonicalHeaderKey]
					if len(v) > 0 {
						return v[0]
					}
				}
				return ""
			}
		}

		// origin.
		if prefix == "origin" {
			if this.origin != nil {
				switch suffix {
				case "address", "addr":
					return this.originAddr
				case "host":
					addr := this.originAddr
					index := strings.Index(addr, ":")
					if index > -1 {
						return addr[:index]
					} else {
						return ""
					}
				case "id":
					return strconv.FormatInt(this.origin.Id, 10)
				case "scheme", "protocol":
					return this.origin.Addr.Protocol.String()
				case "code":
					return this.origin.Code
				}
			}
			return ""
		}

		// node
		if prefix == "node" {
			switch suffix {
			case "id":
				return strconv.FormatInt(this.nodeConfig.Id, 10)
			case "name":
				return this.nodeConfig.Name
			case "role":
				return teaconst.Role
			}
		}

		// host
		if prefix == "host" {
			pieces := strings.Split(this.ReqHost, ".")
			switch suffix {
			case "first":
				if len(pieces) > 0 {
					return pieces[0]
				}
				return ""
			case "last":
				if len(pieces) > 0 {
					return pieces[len(pieces)-1]
				}
				return ""
			case "0":
				if len(pieces) > 0 {
					return pieces[0]
				}
				return ""
			case "1":
				if len(pieces) > 1 {
					return pieces[1]
				}
				return ""
			case "2":
				if len(pieces) > 2 {
					return pieces[2]
				}
				return ""
			case "3":
				if len(pieces) > 3 {
					return pieces[3]
				}
				return ""
			case "4":
				if len(pieces) > 4 {
					return pieces[4]
				}
				return ""
			case "-1":
				if len(pieces) > 0 {
					return pieces[len(pieces)-1]
				}
				return ""
			case "-2":
				if len(pieces) > 1 {
					return pieces[len(pieces)-2]
				}
				return ""
			case "-3":
				if len(pieces) > 2 {
					return pieces[len(pieces)-3]
				}
				return ""
			case "-4":
				if len(pieces) > 3 {
					return pieces[len(pieces)-4]
				}
				return ""
			case "-5":
				if len(pieces) > 4 {
					return pieces[len(pieces)-5]
				}
				return ""
			}
		}

		// geo
		if prefix == "geo" {
			var result = iplib.LookupIP(this.requestRemoteAddr(true))

			switch suffix {
			case "country.name":
				if result != nil && result.IsOk() {
					return result.CountryName()
				}
				return ""
			case "country.id":
				if result != nil && result.IsOk() {
					return types.String(result.CountryId())
				}
				return "0"
			case "province.name":
				if result != nil && result.IsOk() {
					return result.ProvinceName()
				}
				return ""
			case "province.id":
				if result != nil && result.IsOk() {
					return types.String(result.ProvinceId())
				}
				return "0"
			case "city.name":
				if result != nil && result.IsOk() {
					return result.CityName()
				}
				return ""
			case "city.id":
				if result != nil && result.IsOk() {
					return types.String(result.CityId())
				}
				return "0"
			case "town.name":
				if result != nil && result.IsOk() {
					return result.TownName()
				}
				return ""
			case "town.id":
				if result != nil && result.IsOk() {
					return types.String(result.TownId())
				}
				return "0"
			}
		}

		// ips
		if prefix == "isp" {
			var result = iplib.LookupIP(this.requestRemoteAddr(true))

			switch suffix {
			case "name":
				if result != nil && result.IsOk() {
					return result.ProviderName()
				}
			case "id":
				if result != nil && result.IsOk() {
					return types.String(result.ProviderId())
				}
				return "0"
			}
			return ""
		}

		// browser
		if prefix == "browser" {
			var result = stats.SharedUserAgentParser.Parse(this.RawReq.UserAgent())
			switch suffix {
			case "os.name":
				return result.OS.Name
			case "os.version":
				return result.OS.Version
			case "name":
				return result.BrowserName
			case "version":
				return result.BrowserVersion
			case "isMobile":
				if result.IsMobile {
					return "1"
				} else {
					return "0"
				}
			}
		}

		// product
		if prefix == "product" {
			switch suffix {
			case "name":
				if this.nodeConfig.ProductConfig != nil && len(this.nodeConfig.ProductConfig.Name) > 0 {
					return this.nodeConfig.ProductConfig.Name
				}
				return teaconst.GlobalProductName
			case "version":
				if this.nodeConfig.ProductConfig != nil && len(this.nodeConfig.ProductConfig.Version) > 0 {
					return this.nodeConfig.ProductConfig.Version
				}
				return teaconst.Version
			}
		}

		return "${" + varName + "}"
	})
}

// 添加变量定义
func (this *HTTPRequest) addVarMapping(varMapping map[string]string) {
	for k, v := range varMapping {
		this.varMapping[k] = v
	}
}

// 获取请求的客户端地址
func (this *HTTPRequest) requestRemoteAddr(supportVar bool) string {
	if len(this.lnRemoteAddr) > 0 {
		return this.lnRemoteAddr
	}

	if supportVar && len(this.remoteAddr) > 0 {
		return this.remoteAddr
	}

	if supportVar &&
		this.web.RemoteAddr != nil &&
		this.web.RemoteAddr.IsOn &&
		!this.web.RemoteAddr.IsEmpty() {
		if this.web.RemoteAddr.HasValues() { // multiple values
			for _, value := range this.web.RemoteAddr.Values() {
				var remoteAddr = this.Format(value)
				if len(remoteAddr) > 0 && net.ParseIP(remoteAddr) != nil {
					this.remoteAddr = remoteAddr
					return remoteAddr
				}
			}
		} else { // single value
			var remoteAddr = this.Format(this.web.RemoteAddr.Value)
			if len(remoteAddr) > 0 && net.ParseIP(remoteAddr) != nil {
				this.remoteAddr = remoteAddr
				return remoteAddr
			}
		}

		// 如果是从Header中读取，则直接返回原始IP
		if this.web.RemoteAddr.Type == serverconfigs.HTTPRemoteAddrTypeRequestHeader {
			var remoteAddr = this.RawReq.RemoteAddr
			host, _, err := net.SplitHostPort(remoteAddr)
			if err == nil {
				this.remoteAddr = host
				return host
			} else {
				return remoteAddr
			}
		}
	}

	// X-Forwarded-For
	var forwardedFor = this.RawReq.Header.Get("X-Forwarded-For")
	if len(forwardedFor) > 0 {
		commaIndex := strings.Index(forwardedFor, ",")
		if commaIndex > 0 {
			forwardedFor = forwardedFor[:commaIndex]
		}
		if net.ParseIP(forwardedFor) != nil {
			if supportVar {
				this.remoteAddr = forwardedFor
			}
			return forwardedFor
		}
	}

	// Real-IP
	{
		realIP, ok := this.RawReq.Header["X-Real-IP"]
		if ok && len(realIP) > 0 {
			if net.ParseIP(realIP[0]) != nil {
				if supportVar {
					this.remoteAddr = realIP[0]
				}
				return realIP[0]
			}
		}
	}

	// Real-Ip
	{
		realIP, ok := this.RawReq.Header["X-Real-Ip"]
		if ok && len(realIP) > 0 {
			if net.ParseIP(realIP[0]) != nil {
				if supportVar {
					this.remoteAddr = realIP[0]
				}
				return realIP[0]
			}
		}
	}

	// Remote-Addr
	var remoteAddr = this.RawReq.RemoteAddr
	host, _, err := net.SplitHostPort(remoteAddr)
	if err == nil {
		if supportVar {
			this.remoteAddr = host
		}
		return host
	} else {
		return remoteAddr
	}
}

// 获取请求的客户端地址列表
func (this *HTTPRequest) requestRemoteAddrs() (result []string) {
	result = append(result, this.requestRemoteAddr(true))

	// X-Forwarded-For
	var forwardedFor = this.RawReq.Header.Get("X-Forwarded-For")
	if len(forwardedFor) > 0 {
		commaIndex := strings.Index(forwardedFor, ",")
		if commaIndex > 0 && !lists.ContainsString(result, forwardedFor[:commaIndex]) {
			result = append(result, forwardedFor[:commaIndex])
		}
	}

	// Real-IP
	{
		realIP, ok := this.RawReq.Header["X-Real-IP"]
		if ok && len(realIP) > 0 && !lists.ContainsString(result, realIP[0]) {
			result = append(result, realIP[0])
		}
	}

	// Real-Ip
	{
		realIP, ok := this.RawReq.Header["X-Real-Ip"]
		if ok && len(realIP) > 0 && !lists.ContainsString(result, realIP[0]) {
			result = append(result, realIP[0])
		}
	}

	// Remote-Addr
	{
		var remoteAddr = this.RawReq.RemoteAddr
		host, _, err := net.SplitHostPort(remoteAddr)
		if err == nil {
			if !lists.ContainsString(result, host) {
				result = append(result, host)
			}
		} else {
			result = append(result, remoteAddr)
		}
	}

	return
}

// 请求内容长度
func (this *HTTPRequest) requestLength() int64 {
	return this.RawReq.ContentLength
}

// 请求用户
func (this *HTTPRequest) requestRemoteUser() string {
	username, _, ok := this.RawReq.BasicAuth()
	if !ok {
		return ""
	}
	return username
}

// Path 请求的URL中路径部分
func (this *HTTPRequest) Path() string {
	uri, err := url.ParseRequestURI(this.uri)
	if err != nil {
		return ""
	}
	return uri.Path
}

// 客户端端口
func (this *HTTPRequest) requestRemotePort() int {
	_, port, err := net.SplitHostPort(this.RawReq.RemoteAddr)
	if err == nil {
		return types.Int(port)
	}
	return 0
}

// 获取的URI中的参数部分
func (this *HTTPRequest) requestQueryString() string {
	uri, err := url.ParseRequestURI(this.uri)
	if err != nil {
		return ""
	}
	return uri.RawQuery
}

// 构造类似于"GET / HTTP/1.1"之类的请求字符串
func (this *HTTPRequest) requestString() string {
	return this.RawReq.Method + " " + this.rawURI + " " + this.RawReq.Proto
}

// 构造请求字符串
func (this *HTTPRequest) requestCookiesString() string {
	var cookies = []string{}
	for _, cookie := range this.RawReq.Cookies() {
		cookies = append(cookies, url.QueryEscape(cookie.Name)+"="+url.QueryEscape(cookie.Value))
	}
	return strings.Join(cookies, "&")
}

// 查询单个Cookie值
func (this *HTTPRequest) requestCookie(name string) string {
	cookie, err := this.RawReq.Cookie(name)
	if err != nil {
		return ""
	}
	return cookie.Value
}

// 查询请求参数值
func (this *HTTPRequest) requestQueryParam(name string) string {
	uri, err := url.ParseRequestURI(this.rawURI)
	if err != nil {
		return ""
	}

	v, found := uri.Query()[name]
	if !found {
		return ""
	}
	return strings.Join(v, "&")
}

// 查询单个请求Header值
func (this *HTTPRequest) requestHeader(key string) string {
	v, found := this.RawReq.Header[key]
	if !found {
		// 转换为canonical header再尝试
		var canonicalHeaderKey = http.CanonicalHeaderKey(key)
		if canonicalHeaderKey != key {
			return strings.Join(this.RawReq.Header[canonicalHeaderKey], ";")
		}

		return ""
	}
	return strings.Join(v, ";")
}

// 以字符串的形式返回所有请求Header
func (this *HTTPRequest) requestHeadersString() string {
	var headers = []string{}
	for k, v := range this.RawReq.Header {
		for _, subV := range v {
			headers = append(headers, k+": "+subV)
		}
	}
	return strings.Join(headers, ";")
}

// 获取请求Content-Type值
func (this *HTTPRequest) requestContentType() string {
	return this.RawReq.Header.Get("Content-Type")
}

// 获取请求的文件名，仅在请求是读取本地文件时不为空
func (this *HTTPRequest) requestFilename() string {
	return this.filePath
}

// 请求的scheme
func (this *HTTPRequest) requestScheme() string {
	if this.IsHTTPS {
		return "https"
	}
	return "http"
}

// 请求的服务器地址中的端口
func (this *HTTPRequest) requestServerPort() int {
	_, port, err := net.SplitHostPort(this.ServerAddr)
	if err == nil {
		return types.Int(port)
	}
	return 0
}

func (this *HTTPRequest) Id() string {
	return this.requestId
}

func (this *HTTPRequest) Server() maps.Map {
	return maps.Map{"id": this.ReqServer.Id}
}

func (this *HTTPRequest) Node() maps.Map {
	return maps.Map{"id": teaconst.NodeId}
}

// URL 获取完整的URL
func (this *HTTPRequest) URL() string {
	return this.requestScheme() + "://" + this.ReqHost + this.uri
}

// Host 获取Host
func (this *HTTPRequest) Host() string {
	return this.ReqHost
}

func (this *HTTPRequest) Proto() string {
	return this.RawReq.Proto
}

func (this *HTTPRequest) ProtoMajor() int {
	return this.RawReq.ProtoMajor
}

func (this *HTTPRequest) ProtoMinor() int {
	return this.RawReq.ProtoMinor
}

func (this *HTTPRequest) RemoteAddr() string {
	return this.requestRemoteAddr(true)
}

func (this *HTTPRequest) RawRemoteAddr() string {
	var addr = this.RawReq.RemoteAddr
	host, _, err := net.SplitHostPort(addr)
	if err == nil {
		addr = host
	}
	return addr
}

func (this *HTTPRequest) RemotePort() int {
	addr := this.RawReq.RemoteAddr
	_, port, err := net.SplitHostPort(addr)
	if err != nil {
		return 0
	}
	return types.Int(port)
}

func (this *HTTPRequest) SetAttr(name string, value string) {
	this.logAttrs[name] = value
}

func (this *HTTPRequest) SetVar(name string, value string) {
	this.varMapping[name] = value
}

// ContentLength 请求内容长度
func (this *HTTPRequest) ContentLength() int64 {
	return this.RawReq.ContentLength
}

// CalculateSize 计算当前请求的尺寸（预估）
func (this *HTTPRequest) CalculateSize() (size int64) {
	// Get /xxx HTTP/1.1
	size += int64(len(this.RawReq.Method)) + 1
	size += int64(len(this.RawReq.URL.String())) + 1
	size += int64(len(this.RawReq.Proto)) + 1
	for k, v := range this.RawReq.Header {
		for _, v1 := range v {
			size += int64(len(k) + 2 /** : **/ + len(v1) + 1)
		}
	}

	size += 1 /** \r\n **/

	if this.RawReq.ContentLength > 0 {
		size += this.RawReq.ContentLength
	} else if len(this.requestBodyData) > 0 {
		size += int64(len(this.requestBodyData))
	}
	return size
}

// Method 请求方法
func (this *HTTPRequest) Method() string {
	return this.RawReq.Method
}

// TransferEncoding 获取传输编码
func (this *HTTPRequest) TransferEncoding() string {
	if len(this.RawReq.TransferEncoding) > 0 {
		return this.RawReq.TransferEncoding[0]
	}
	return ""
}

// Cookie 获取Cookie
func (this *HTTPRequest) Cookie(name string) string {
	c, err := this.RawReq.Cookie(name)
	if err != nil {
		return ""
	}
	return c.Value
}

// DeleteHeader 删除请求Header
func (this *HTTPRequest) DeleteHeader(name string) {
	this.RawReq.Header.Del(name)
}

// SetHeader 设置请求Header
func (this *HTTPRequest) SetHeader(name string, values []string) {
	this.RawReq.Header[name] = values
}

// Header 读取Header
func (this *HTTPRequest) Header() http.Header {
	return this.RawReq.Header
}

// URI 获取当前请求的URI
func (this *HTTPRequest) URI() string {
	return this.uri
}

// SetURI 设置当前请求的URI
func (this *HTTPRequest) SetURI(uri string) {
	this.uri = uri
}

// Done 设置已完成
func (this *HTTPRequest) Done() {
	this.isDone = true
}

// Close 关闭连接
func (this *HTTPRequest) Close() {
	this.Done()

	var requestConn = this.RawReq.Context().Value(HTTPConnContextKey)
	if requestConn == nil {
		return
	}

	lingerConn, ok := requestConn.(LingerConn)
	if ok {
		_ = lingerConn.SetLinger(0)
	}

	conn, ok := requestConn.(net.Conn)
	if ok {
		_ = conn.Close()
		return
	}
}

// Allow 放行
func (this *HTTPRequest) Allow() {
	this.web.FirewallRef = nil
}

// 设置代理相关头部信息
// 参考：https://tools.ietf.org/html/rfc7239
func (this *HTTPRequest) setForwardHeaders(header http.Header) {
	// TODO 做成可选项
	if this.RawReq.Header.Get("Connection") == "close" {
		this.RawReq.Header.Set("Connection", "keep-alive")
	}

	var rawRemoteAddr = this.RawReq.RemoteAddr
	host, _, err := net.SplitHostPort(rawRemoteAddr)
	if err == nil {
		rawRemoteAddr = host
	}

	// x-real-ip
	if this.reverseProxy != nil && this.reverseProxy.ShouldAddXRealIPHeader() {
		_, ok1 := header["X-Real-IP"]
		_, ok2 := header["X-Real-Ip"]
		if !ok1 && !ok2 {
			header["X-Real-IP"] = []string{this.requestRemoteAddr(true)}
		}
	}

	// X-Forwarded-For
	if this.reverseProxy != nil && this.reverseProxy.ShouldAddXForwardedForHeader() {
		forwardedFor, ok := header["X-Forwarded-For"]
		if ok && len(forwardedFor) > 0 { // already exists
			_, hasForwardHeader := this.RawReq.Header["X-Forwarded-For"]
			if hasForwardHeader {
				header["X-Forwarded-For"] = []string{strings.Join(forwardedFor, ", ") + ", " + rawRemoteAddr}
			}
		} else {
			var clientRemoteAddr = this.requestRemoteAddr(true)
			if len(clientRemoteAddr) > 0 && clientRemoteAddr != rawRemoteAddr {
				header["X-Forwarded-For"] = []string{clientRemoteAddr + ", " + rawRemoteAddr}
			} else {
				header["X-Forwarded-For"] = []string{rawRemoteAddr}
			}
		}
	}

	// Forwarded
	/**{
		forwarded, ok := header["Forwarded"]
		if ok {
			header["Forwarded"] = []string{strings.Join(forwarded, ", ") + ", by=" + this.serverAddr + "; for=" + remoteAddr + "; host=" + this.ReqHost + "; proto=" + this.rawScheme}
		} else {
			header["Forwarded"] = []string{"by=" + this.serverAddr + "; for=" + remoteAddr + "; host=" + this.ReqHost + "; proto=" + this.rawScheme}
		}
	}**/

	// others
	if this.reverseProxy != nil && this.reverseProxy.ShouldAddXForwardedByHeader() {
		this.RawReq.Header.Set("X-Forwarded-By", this.ServerAddr)
	}

	if this.reverseProxy != nil && this.reverseProxy.ShouldAddXForwardedHostHeader() {
		if _, ok := header["X-Forwarded-Host"]; !ok {
			this.RawReq.Header.Set("X-Forwarded-Host", this.ReqHost)
		}
	}

	if this.reverseProxy != nil && this.reverseProxy.ShouldAddXForwardedProtoHeader() {
		if _, ok := header["X-Forwarded-Proto"]; !ok {
			this.RawReq.Header.Set("X-Forwarded-Proto", this.requestScheme())
		}
	}
}

// 处理自定义Request Header
func (this *HTTPRequest) processRequestHeaders(reqHeader http.Header) {
	this.fixRequestHeader(reqHeader)

	if this.web.RequestHeaderPolicy != nil && this.web.RequestHeaderPolicy.IsOn {
		// 删除某些Header
		for name := range reqHeader {
			if this.web.RequestHeaderPolicy.ContainsDeletedHeader(name) {
				reqHeader.Del(name)
			}
		}

		// Set
		for _, header := range this.web.RequestHeaderPolicy.SetHeaders {
			if !header.IsOn {
				continue
			}

			// 是否已删除
			if this.web.RequestHeaderPolicy.ContainsDeletedHeader(header.Name) {
				continue
			}

			// 请求方法
			if len(header.Methods) > 0 && !lists.ContainsString(header.Methods, this.RawReq.Method) {
				continue
			}

			// 域名
			if len(header.Domains) > 0 && !configutils.MatchDomains(header.Domains, this.ReqHost) {
				continue
			}

			var headerValue = header.Value
			if header.ShouldReplace {
				if len(headerValue) == 0 {
					headerValue = reqHeader.Get(header.Name) // 原有值
				} else if header.HasVariables() {
					headerValue = this.Format(header.Value)
				}

				for _, v := range header.ReplaceValues {
					headerValue = v.Replace(headerValue)
				}
			} else if header.HasVariables() {
				headerValue = this.Format(header.Value)
			}

			// 支持修改Host
			if header.Name == "Host" && len(header.Value) > 0 {
				this.RawReq.Host = headerValue
			} else {
				if header.ShouldAppend {
					reqHeader[header.Name] = append(reqHeader[header.Name], headerValue)
				} else {
					reqHeader[header.Name] = []string{headerValue}
				}
			}
		}

		// 非标Header
		if len(this.web.RequestHeaderPolicy.NonStandardHeaders) > 0 {
			for _, name := range this.web.RequestHeaderPolicy.NonStandardHeaders {
				var canonicalKey = http.CanonicalHeaderKey(name)
				if canonicalKey != name {
					v, ok := reqHeader[canonicalKey]
					if ok {
						delete(reqHeader, canonicalKey)
						reqHeader[name] = v
					}
				}
			}
		}
	}
}

// 处理一些被Golang转换了的Header
// TODO 可以自定义要转换的Header
func (this *HTTPRequest) fixRequestHeader(header http.Header) {
	for k, v := range header {
		if strings.Contains(k, "-Websocket-") {
			header.Del(k)
			k = strings.ReplaceAll(k, "-Websocket-", "-WebSocket-")
			header[k] = v
		} else if strings.HasPrefix(k, "Sec-Ch") {
			header.Del(k)
			k = strings.ReplaceAll(k, "Sec-Ch-Ua", "Sec-CH-UA")
			header[k] = v
		} else {
			switch k {
			case "Www-Authenticate":
				header.Del(k)
				header["WWW-Authenticate"] = v
			case "A-Im":
				header.Del(k)
				header["A-IM"] = v
			case "Content-Md5":
				header.Del(k)
				header["Content-MD5"] = v
			case "Sec-Gpc":
				header.Del(k)
				header["Content-GPC"] = v
			}
		}
	}
}

// ProcessResponseHeaders 处理自定义Response Header
func (this *HTTPRequest) ProcessResponseHeaders(responseHeader http.Header, statusCode int) {
	// Server Name
	if this.nodeConfig != nil && this.nodeConfig.GlobalServerConfig != nil && len(this.nodeConfig.GlobalServerConfig.HTTPAll.ServerName) > 0 {
		responseHeader.Set("Server", this.nodeConfig.GlobalServerConfig.HTTPAll.ServerName)
	}

	// 删除/添加/替换Header
	// TODO 实现AddTrailers
	if this.web.ResponseHeaderPolicy != nil && this.web.ResponseHeaderPolicy.IsOn {
		// 删除某些Header
		for name := range responseHeader {
			if this.web.ResponseHeaderPolicy.ContainsDeletedHeader(name) {
				responseHeader.Del(name)
			}
		}

		// Set
		for _, header := range this.web.ResponseHeaderPolicy.SetHeaders {
			if !header.IsOn {
				continue
			}

			// 是否已删除
			if this.web.ResponseHeaderPolicy.ContainsDeletedHeader(header.Name) {
				continue
			}

			// 状态码
			if header.Status != nil && !header.Status.Match(statusCode) {
				continue
			}

			// 请求方法
			if len(header.Methods) > 0 && !lists.ContainsString(header.Methods, this.RawReq.Method) {
				continue
			}

			// 域名
			if len(header.Domains) > 0 && !configutils.MatchDomains(header.Domains, this.ReqHost) {
				continue
			}

			// 是否为跳转
			if header.DisableRedirect && httpStatusIsRedirect(statusCode) {
				continue
			}

			var headerValue = header.Value
			if header.ShouldReplace {
				if len(headerValue) == 0 {
					headerValue = responseHeader.Get(header.Name) // 原有值
				} else if header.HasVariables() {
					headerValue = this.Format(header.Value)
				}

				for _, v := range header.ReplaceValues {
					headerValue = v.Replace(headerValue)
				}
			} else if header.HasVariables() {
				headerValue = this.Format(header.Value)
			}

			if header.ShouldAppend {
				responseHeader[header.Name] = append(responseHeader[header.Name], headerValue)
			} else {
				responseHeader[header.Name] = []string{headerValue}
			}
		}

		// 非标Header
		if len(this.web.ResponseHeaderPolicy.NonStandardHeaders) > 0 {
			for _, name := range this.web.ResponseHeaderPolicy.NonStandardHeaders {
				var canonicalKey = http.CanonicalHeaderKey(name)
				if canonicalKey != name {
					v, ok := responseHeader[canonicalKey]
					if ok {
						delete(responseHeader, canonicalKey)
						responseHeader[name] = v
					}
				}
			}
		}

		// CORS
		if this.web.ResponseHeaderPolicy.CORS != nil && this.web.ResponseHeaderPolicy.CORS.IsOn && (!this.web.ResponseHeaderPolicy.CORS.OptionsMethodOnly || this.RawReq.Method == http.MethodOptions) {
			var corsConfig = this.web.ResponseHeaderPolicy.CORS

			// Allow-Origin
			if len(corsConfig.AllowOrigin) == 0 {
				var origin = this.RawReq.Header.Get("Origin")
				if len(origin) > 0 {
					responseHeader.Set("Access-Control-Allow-Origin", origin)
				}
			} else {
				responseHeader.Set("Access-Control-Allow-Origin", corsConfig.AllowOrigin)
			}

			// Allow-Methods
			if len(corsConfig.AllowMethods) == 0 {
				responseHeader.Set("Access-Control-Allow-Methods", "PUT, GET, POST, DELETE, HEAD, OPTIONS, PATCH")
			} else {
				responseHeader.Set("Access-Control-Allow-Methods", strings.Join(corsConfig.AllowMethods, ", "))
			}

			// Max-Age
			if corsConfig.MaxAge > 0 {
				responseHeader.Set("Access-Control-Max-Age", types.String(corsConfig.MaxAge))
			}

			// Expose-Headers
			if len(corsConfig.ExposeHeaders) > 0 {
				responseHeader.Set("Access-Control-Expose-Headers", strings.Join(corsConfig.ExposeHeaders, ", "))
			}

			// Request-Method
			if len(corsConfig.RequestMethod) > 0 {
				responseHeader.Set("Access-Control-Request-Method", strings.ToUpper(corsConfig.RequestMethod))
			}

			// Allow-Credentials
			responseHeader.Set("Access-Control-Allow-Credentials", "true")
		}
	}

	// HSTS
	if this.IsHTTPS &&
		this.ReqServer.HTTPS != nil &&
		this.ReqServer.HTTPS.SSLPolicy != nil &&
		this.ReqServer.HTTPS.SSLPolicy.IsOn &&
		this.ReqServer.HTTPS.SSLPolicy.HSTS != nil &&
		this.ReqServer.HTTPS.SSLPolicy.HSTS.IsOn &&
		this.ReqServer.HTTPS.SSLPolicy.HSTS.Match(this.ReqHost) {
		responseHeader.Set(this.ReqServer.HTTPS.SSLPolicy.HSTS.HeaderKey(), this.ReqServer.HTTPS.SSLPolicy.HSTS.HeaderValue())
	}

	// HTTP/3
	if this.IsHTTPS && !this.IsHTTP3 && this.ReqServer.SupportsHTTP3() {
		this.processHTTP3Headers(responseHeader)
	}
}

// 添加错误信息
func (this *HTTPRequest) addError(err error) {
	if err == nil {
		return
	}
	this.errors = append(this.errors, err.Error())
}

// 计算合适的buffer size
func (this *HTTPRequest) bytePool(contentLength int64) *utils.BytePool {
	if contentLength < 0 {
		return utils.BytePool16k
	}
	if contentLength < 8192 { // 8K
		return utils.BytePool1k
	}
	if contentLength < 32768 { // 32K
		return utils.BytePool16k
	}
	if contentLength < 131072 { // 128K
		return utils.BytePool32k
	}
	return utils.BytePool32k
}

// 检查是否可以忽略错误
func (this *HTTPRequest) canIgnore(err error) bool {
	if err == nil {
		return true
	}

	// 已读到头
	if err == io.EOF || err == io.ErrUnexpectedEOF {
		return true
	}

	// 网络错误
	_, ok := err.(*net.OpError)
	if ok {
		return true
	}

	// 客户端主动取消
	if err == errWritingToClient ||
		err == context.Canceled ||
		err == io.ErrShortWrite ||
		strings.Contains(err.Error(), "write: connection") ||
		strings.Contains(err.Error(), "write: broken pipe") ||
		strings.Contains(err.Error(), "write tcp") {
		return true
	}

	// HTTP/2流错误
	if err.Error() == "http2: stream closed" || strings.Contains(err.Error(), "stream error") || err.Error() == "client disconnected" { // errStreamClosed, errClientDisconnected
		return true
	}

	// HTTP内部错误
	if strings.HasPrefix(err.Error(), "http:") || strings.HasPrefix(err.Error(), "http2:") {
		return true
	}

	return false
}

// 检查连接是否已关闭
func (this *HTTPRequest) isConnClosed() bool {
	var requestConn = this.RawReq.Context().Value(HTTPConnContextKey)
	if requestConn == nil {
		return true
	}

	conn, ok := requestConn.(net.Conn)
	if ok {
		return isClientConnClosed(conn)
	}

	return true
}
