package nodes

import (
	"context"
	"errors"
	"fmt"
	"github.com/TeaOSLab/EdgeCommon/pkg/configutils"
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs"
	teaconst "github.com/TeaOSLab/EdgeNode/internal/const"
	"github.com/TeaOSLab/EdgeNode/internal/stats"
	"github.com/TeaOSLab/EdgeNode/internal/utils"
	"github.com/iwind/TeaGo/types"
	"golang.org/x/net/http2"
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

// byte pool
var bytePool256b = utils.NewBytePool(20480, 256)
var bytePool1k = utils.NewBytePool(20480, 1024)
var bytePool32k = utils.NewBytePool(20480, 32*1024)
var bytePool128k = utils.NewBytePool(20480, 128*1024)

// HTTPRequest HTTP请求
type HTTPRequest struct {
	// 外部参数
	RawReq     *http.Request
	RawWriter  http.ResponseWriter
	Server     *serverconfigs.ServerConfig
	Host       string // 请求的Host
	ServerName string // 实际匹配到的Host
	ServerAddr string // 实际启动的服务器监听地址
	IsHTTP     bool
	IsHTTPS    bool

	// 内部参数
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
	errors               []string                          // 错误信息
	rewriteRule          *serverconfigs.HTTPRewriteRule    // 匹配到的重写规则
	rewriteReplace       string                            // 重写规则的目标
	rewriteIsExternalURL bool                              // 重写目标是否为外部URL
	cacheRef             *serverconfigs.HTTPCacheRef       // 缓存设置
	cacheKey             string                            // 缓存使用的Key
	isCached             bool                              // 是否已经被缓存

	// WAF相关
	firewallPolicyId    int64
	firewallRuleGroupId int64
	firewallRuleSetId   int64
	firewallRuleId      int64

	logAttrs map[string]string

	disableLog bool // 此请求中关闭Log
}

// 初始化
func (this *HTTPRequest) init() {
	this.writer = NewHTTPWriter(this, this.RawWriter)
	this.web = &serverconfigs.HTTPWebConfig{IsOn: true}
	// this.uri = this.RawReq.URL.RequestURI()
	// 之所以不使用RequestURI()，是不想让URL中的Path被Encode
	if len(this.RawReq.URL.RawQuery) > 0 {
		this.uri = this.RawReq.URL.Path + "?" + this.RawReq.URL.RawQuery
	} else {
		this.uri = this.RawReq.URL.Path
	}

	this.rawURI = this.uri
	this.varMapping = map[string]string{
		// 缓存相关初始化
		"cache.status":      "BYPASS",
		"cache.policy.name": "",
		"cache.policy.id":   "0",
		"cache.policy.type": "",
	}
	this.logAttrs = map[string]string{}
	this.requestFromTime = time.Now()
}

// Do 执行请求
func (this *HTTPRequest) Do() {
	// 初始化
	this.init()

	// 当前服务的反向代理配置
	if this.Server.ReverseProxyRef != nil && this.Server.ReverseProxy != nil {
		this.reverseProxyRef = this.Server.ReverseProxyRef
		this.reverseProxy = this.Server.ReverseProxy
	}

	// Web配置
	err := this.configureWeb(this.Server.Web, true, 0)
	if err != nil {
		this.write500(err)
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

	// 访问控制
	// TODO 需要实现

	// 自动跳转到HTTPS
	if this.IsHTTP && this.web.RedirectToHttps != nil && this.web.RedirectToHttps.IsOn {
		this.doRedirectToHTTPS(this.web.RedirectToHttps)
		this.doEnd()
		return
	}

	// Gzip
	if this.web.GzipRef != nil && this.web.GzipRef.IsOn && this.web.Gzip != nil && this.web.Gzip.IsOn && this.web.Gzip.Level > 0 {
		this.writer.Gzip(this.web.Gzip)
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
	// 统计
	if this.web.StatRef != nil && this.web.StatRef.IsOn {
		this.doStat()
	}

	// 跳转
	if len(this.web.HostRedirects) > 0 {
		if this.doHostRedirect() {
			return
		}
	}

	// 特殊URL处理
	if len(this.rawURI) > 1 && this.rawURI[1] == '.' {
		// ACME
		// TODO 需要配置是否启用ACME检测
		if strings.HasPrefix(this.rawURI, "/.well-known/acme-challenge/") {
			this.doACME()
			return
		}
	}

	// 临时关闭页面
	if this.web.Shutdown != nil && this.web.Shutdown.IsOn {
		this.doShutdown()
		return
	}

	// 缓存
	if this.web.Cache != nil && this.web.Cache.IsOn {
		if this.doCacheRead() {
			return
		}
	}

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
	if this.Server != nil {
		if this.isCached {
			stats.SharedTrafficStatManager.Add(this.Server.Id, this.writer.sentBodyBytes, this.writer.sentBodyBytes, 1, 1)
		} else {
			stats.SharedTrafficStatManager.Add(this.Server.Id, this.writer.sentBodyBytes, 0, 1, 0)
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

	// charset
	if web.Charset != nil && (web.Charset.IsPrior || isTop) {
		this.web.Charset = web.Charset
	}

	// websocket
	if web.WebsocketRef != nil && (web.WebsocketRef.IsPrior || isTop) {
		this.web.WebsocketRef = web.WebsocketRef
		this.web.Websocket = web.Websocket
	}

	// gzip
	if web.GzipRef != nil && (web.GzipRef.IsPrior || isTop) {
		this.web.GzipRef = web.GzipRef
		this.web.Gzip = web.Gzip
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
			return this.requestRemoteAddr()
		case "rawRemoteAddr":
			addr := this.RawReq.RemoteAddr
			host, _, err := net.SplitHostPort(addr)
			if err == nil {
				addr = host
			}
			return addr
		case "remotePort":
			return strconv.Itoa(this.requestRemotePort())
		case "remoteUser":
			return this.requestRemoteUser()
		case "requestURI", "requestUri":
			return this.rawURI
		case "requestPath":
			return this.requestPath()
		case "requestPathExtension":
			return filepath.Ext(this.requestPath())
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
				return filepath.Clean(this.web.Root.Dir + this.requestPath())
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
			return this.Host
		case "referer":
			return this.RawReq.Referer()
		case "userAgent":
			return this.RawReq.UserAgent()
		case "contentType":
			return this.requestContentType()
		case "request":
			return this.requestString()
		case "cookies":
			return this.requestCookiesString()
		case "args", "queryString":
			return this.requestQueryString()
		case "headers":
			return this.requestHeadersString()
		case "serverName":
			return this.ServerName
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
			dotIndex := strings.Index(suffix, ".")
			if dotIndex < 0 {
				return "${" + varName + "}"
			}
			switch suffix[:dotIndex] {
			case "header":
				return this.writer.Header().Get(suffix[dotIndex+1:])
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
				return strconv.FormatInt(sharedNodeConfig.Id, 10)
			case "name":
				return sharedNodeConfig.Name
			case "role":
				return teaconst.Role
			}
		}

		// host
		if prefix == "host" {
			pieces := strings.Split(this.Host, ".")
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
func (this *HTTPRequest) requestRemoteAddr() string {
	// X-Forwarded-For
	forwardedFor := this.RawReq.Header.Get("X-Forwarded-For")
	if len(forwardedFor) > 0 {
		commaIndex := strings.Index(forwardedFor, ",")
		if commaIndex > 0 {
			return forwardedFor[:commaIndex]
		}
		return forwardedFor
	}

	// Real-IP
	{
		realIP, ok := this.RawReq.Header["X-Real-IP"]
		if ok && len(realIP) > 0 {
			return realIP[0]
		}
	}

	// Real-Ip
	{
		realIP, ok := this.RawReq.Header["X-Real-Ip"]
		if ok && len(realIP) > 0 {
			return realIP[0]
		}
	}

	// Remote-Addr
	remoteAddr := this.RawReq.RemoteAddr
	host, _, err := net.SplitHostPort(remoteAddr)
	if err == nil {
		return host
	} else {
		return remoteAddr
	}
}

// 获取请求的客户端地址列表
func (this *HTTPRequest) requestRemoteAddrs() (result []string) {
	// X-Forwarded-For
	forwardedFor := this.RawReq.Header.Get("X-Forwarded-For")
	if len(forwardedFor) > 0 {
		commaIndex := strings.Index(forwardedFor, ",")
		if commaIndex > 0 {
			result = append(result, forwardedFor[:commaIndex])
		}
	}

	// Real-IP
	{
		realIP, ok := this.RawReq.Header["X-Real-IP"]
		if ok && len(realIP) > 0 {
			result = append(result, realIP[0])
		}
	}

	// Real-Ip
	{
		realIP, ok := this.RawReq.Header["X-Real-Ip"]
		if ok && len(realIP) > 0 {
			result = append(result, realIP[0])
		}
	}

	// Remote-Addr
	remoteAddr := this.RawReq.RemoteAddr
	host, _, err := net.SplitHostPort(remoteAddr)
	if err == nil {
		result = append(result, host)
	} else {
		result = append(result, remoteAddr)
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

// 请求的URL中路径部分
func (this *HTTPRequest) requestPath() string {
	uri, err := url.ParseRequestURI(this.rawURI)
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

// 情趣的URI中的参数部分
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

// 设置代理相关头部信息
// 参考：https://tools.ietf.org/html/rfc7239
func (this *HTTPRequest) setForwardHeaders(header http.Header) {
	// TODO 做成可选项
	if this.RawReq.Header.Get("Connection") == "close" {
		this.RawReq.Header.Set("Connection", "keep-alive")
	}

	remoteAddr := this.RawReq.RemoteAddr
	host, _, err := net.SplitHostPort(remoteAddr)
	if err == nil {
		remoteAddr = host
	}

	// x-real-ip
	if this.reverseProxy != nil && this.reverseProxy.ShouldAddXRealIPHeader() {
		_, ok1 := header["X-Real-IP"]
		_, ok2 := header["X-Real-Ip"]
		if !ok1 && !ok2 {
			header["X-Real-IP"] = []string{remoteAddr}
		}
	}

	// X-Forwarded-For
	if this.reverseProxy != nil && this.reverseProxy.ShouldAddXForwardedForHeader() {
		forwardedFor, ok := header["X-Forwarded-For"]
		if ok {
			_, hasForwardHeader := this.RawReq.Header["X-Forwarded-For"]
			if hasForwardHeader {
				header["X-Forwarded-For"] = []string{strings.Join(forwardedFor, ", ") + ", " + remoteAddr}
			}
		} else {
			header["X-Forwarded-For"] = []string{remoteAddr}
		}
	}

	// Forwarded
	/**{
		forwarded, ok := header["Forwarded"]
		if ok {
			header["Forwarded"] = []string{strings.Join(forwarded, ", ") + ", by=" + this.serverAddr + "; for=" + remoteAddr + "; host=" + this.host + "; proto=" + this.rawScheme}
		} else {
			header["Forwarded"] = []string{"by=" + this.serverAddr + "; for=" + remoteAddr + "; host=" + this.host + "; proto=" + this.rawScheme}
		}
	}**/

	// others
	if this.reverseProxy != nil && this.reverseProxy.ShouldAddXForwardedByHeader() {
		this.RawReq.Header.Set("X-Forwarded-By", this.ServerAddr)
	}

	if this.reverseProxy != nil && this.reverseProxy.ShouldAddXForwardedHostHeader() {
		if _, ok := header["X-Forwarded-Host"]; !ok {
			this.RawReq.Header.Set("X-Forwarded-Host", this.Host)
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

		// Add
		for _, header := range this.web.RequestHeaderPolicy.AddHeaders {
			if !header.IsOn {
				continue
			}
			oldValues, _ := this.RawReq.Header[header.Name]
			newHeaderValue := header.Value // 因为我们不能修改header，所以在这里使用新变量
			if header.HasVariables() {
				newHeaderValue = this.Format(header.Value)
			}
			oldValues = append(oldValues, newHeaderValue)
			reqHeader[header.Name] = oldValues

			// 支持修改Host
			if header.Name == "Host" && len(header.Value) > 0 {
				this.RawReq.Host = newHeaderValue
			}
		}

		// Set
		for _, header := range this.web.RequestHeaderPolicy.SetHeaders {
			if !header.IsOn {
				continue
			}
			newHeaderValue := header.Value // 因为我们不能修改header，所以在这里使用新变量
			if header.HasVariables() {
				newHeaderValue = this.Format(header.Value)
			}
			reqHeader[header.Name] = []string{newHeaderValue}

			// 支持修改Host
			if header.Name == "Host" && len(header.Value) > 0 {
				this.RawReq.Host = newHeaderValue
			}
		}

		// Replace
		// TODO 需要实现
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
		} else if k == "Www-Authenticate" {
			header.Del(k)
			header["WWW-Authenticate"] = v
		}
	}
}

// 处理自定义Response Header
func (this *HTTPRequest) processResponseHeaders(statusCode int) {
	responseHeader := this.writer.Header()

	// 删除/添加/替换Header
	// TODO 实现AddTrailers
	// TODO 实现ReplaceHeaders
	if this.web.ResponseHeaderPolicy != nil && this.web.ResponseHeaderPolicy.IsOn {
		// 删除某些Header
		for name := range responseHeader {
			if this.web.ResponseHeaderPolicy.ContainsDeletedHeader(name) {
				responseHeader.Del(name)
			}
		}

		// Add
		for _, header := range this.web.ResponseHeaderPolicy.AddHeaders {
			if !header.IsOn {
				continue
			}
			if header.Match(statusCode) {
				if this.web.ResponseHeaderPolicy.ContainsDeletedHeader(header.Name) {
					continue
				}
				oldValues, _ := responseHeader[header.Name]
				if header.HasVariables() {
					oldValues = append(oldValues, this.Format(header.Value))
				} else {
					oldValues = append(oldValues, header.Value)
				}
				responseHeader[header.Name] = oldValues
			}
		}

		// Set
		for _, header := range this.web.ResponseHeaderPolicy.SetHeaders {
			if !header.IsOn {
				continue
			}
			if header.Match(statusCode) {
				if this.web.ResponseHeaderPolicy.ContainsDeletedHeader(header.Name) {
					continue
				}
				if header.HasVariables() {
					responseHeader[header.Name] = []string{this.Format(header.Value)}
				} else {
					responseHeader[header.Name] = []string{header.Value}
				}
			}
		}

		// Replace
		// TODO
	}

	// HSTS
	if this.IsHTTPS &&
		this.Server.HTTPS != nil &&
		this.Server.HTTPS.SSLPolicy != nil &&
		this.Server.HTTPS.SSLPolicy.IsOn &&
		this.Server.HTTPS.SSLPolicy.HSTS != nil &&
		this.Server.HTTPS.SSLPolicy.HSTS.IsOn &&
		this.Server.HTTPS.SSLPolicy.HSTS.Match(this.Host) {
		responseHeader.Set(this.Server.HTTPS.SSLPolicy.HSTS.HeaderKey(), this.Server.HTTPS.SSLPolicy.HSTS.HeaderValue())
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
	if contentLength <= 0 {
		return bytePool1k
	}
	if contentLength < 1024 { // 1K
		return bytePool256b
	}
	if contentLength < 32768 { // 32K
		return bytePool1k
	}
	if contentLength < 1048576 { // 1M
		return bytePool32k
	}
	return bytePool128k
}

// 检查是否可以忽略错误
func (this *HTTPRequest) canIgnore(err error) bool {
	if err == nil {
		return true
	}

	// 网络错误
	_, ok := err.(*net.OpError)
	if ok {
		return true
	}

	// 客户端主动取消
	if err == context.Canceled || err == io.ErrShortWrite {
		return true
	}

	// HTTP/2流错误
	{
		_, ok := err.(http2.StreamError)
		if ok {
			return true
		}
	}
	return false
}
