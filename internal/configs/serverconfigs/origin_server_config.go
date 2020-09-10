package serverconfigs

import (
	"crypto/tls"
	"errors"
	"fmt"
	"github.com/TeaOSLab/EdgeNode/internal/configs/serverconfigs/shared"
	"github.com/TeaOSLab/EdgeNode/internal/configs/serverconfigs/sslconfigs"
	"net"
	"strconv"
	"strings"
	"time"
)

// 源站服务配置
type OriginServerConfig struct {
	HeaderList *shared.HeaderList `yaml:"headers" json:"headers"`

	Id          int64                 `yaml:"id" json:"id"`                   // ID
	IsOn        bool                  `yaml:"isOn" json:"isOn"`               // 是否启用 TODO
	Version     int                   `yaml:"version" json:"version"`         // 版本
	Name        string                `yaml:"name" json:"name"`               // 名称 TODO
	Addr        *NetworkAddressConfig `yaml:"addr" json:"addr"`               // 地址
	Description string                `yaml:"description" json:"description"` // 描述 TODO
	Code        string                `yaml:"code" json:"code"`               // 代号 TODO
	Scheme      string                `yaml:"scheme" json:"scheme"`           // 协议 TODO

	Weight       uint   `yaml:"weight" json:"weight"`             // 权重 TODO
	IsBackup     bool   `yaml:"backup" json:"isBackup"`           // 是否为备份 TODO
	FailTimeout  string `yaml:"failTimeout" json:"failTimeout"`   // 连接失败超时 TODO
	ReadTimeout  string `yaml:"readTimeout" json:"readTimeout"`   // 读取超时时间 TODO
	IdleTimeout  string `yaml:"idleTimeout" json:"idleTimeout"`   // 空闲连接超时时间 TODO
	MaxFails     int32  `yaml:"maxFails" json:"maxFails"`         // 最多失败次数 TODO
	CurrentFails int32  `yaml:"currentFails" json:"currentFails"` // 当前已失败次数 TODO
	MaxConns     int32  `yaml:"maxConns" json:"maxConns"`         // 最大并发连接数 TODO
	CurrentConns int32  `yaml:"currentConns" json:"currentConns"` // 当前连接数 TODO
	IdleConns    int32  `yaml:"idleConns" json:"idleConns"`       // 最大空闲连接数 TODO

	IsDown   bool      `yaml:"down" json:"isDown"`                           // 是否下线 TODO
	DownTime time.Time `yaml:"downTime,omitempty" json:"downTime,omitempty"` // 下线时间 TODO

	RequestURI      string                 `yaml:"requestURI" json:"requestURI"`           // 转发后的请求URI TODO
	ResponseHeaders []*shared.HeaderConfig `yaml:"responseHeaders" json:"responseHeaders"` // 响应Header TODO
	Host            string                 `yaml:"host" json:"host"`                       // 自定义主机名 TODO

	// 健康检查URL，目前支持：
	// - http|https 返回2xx-3xx认为成功
	HealthCheck struct {
		IsOn        bool                `yaml:"isOn" json:"isOn"`               // 是否开启 TODO
		URL         string              `yaml:"url" json:"url"`                 // TODO
		Interval    int                 `yaml:"interval" json:"interval"`       // TODO
		StatusCodes []int               `yaml:"statusCodes" json:"statusCodes"` // TODO
		Timeout     shared.TimeDuration `yaml:"timeout" json:"timeout"`         // 超时时间 TODO
	} `yaml:"healthCheck" json:"healthCheck"`

	Cert *sslconfigs.SSLCertConfig `yaml:"cert" json:"cert"` // 请求源服务器用的证书

	// ftp
	FTP *OriginServerFTPConfig `yaml:"ftp" json:"ftp"`

	failTimeoutDuration time.Duration
	readTimeoutDuration time.Duration
	idleTimeoutDuration time.Duration

	hasRequestURI bool
	requestPath   string
	requestArgs   string

	hasRequestHeaders  bool
	hasResponseHeaders bool

	hasHost bool

	uniqueKey string

	hasAddrVariables bool // 地址中是否含有变量
}

// 校验
func (this *OriginServerConfig) Init() error {
	// 证书
	if this.Cert != nil {
		err := this.Cert.Init()
		if err != nil {
			return err
		}
	}

	// unique key
	this.uniqueKey = strconv.FormatInt(this.Id, 10) + "@" + fmt.Sprintf("%d", this.Version)

	// failTimeout
	if len(this.FailTimeout) > 0 {
		this.failTimeoutDuration, _ = time.ParseDuration(this.FailTimeout)
	}

	// readTimeout
	if len(this.ReadTimeout) > 0 {
		this.readTimeoutDuration, _ = time.ParseDuration(this.ReadTimeout)
	}

	// idleTimeout
	if len(this.IdleTimeout) > 0 {
		this.idleTimeoutDuration, _ = time.ParseDuration(this.IdleTimeout)
	}

	// Headers
	if this.HeaderList != nil {
		err := this.HeaderList.Init()
		if err != nil {
			return err
		}
	}

	// request uri
	if len(this.RequestURI) == 0 || this.RequestURI == "${requestURI}" {
		this.hasRequestURI = false
	} else {
		this.hasRequestURI = true

		if strings.Contains(this.RequestURI, "?") {
			pieces := strings.SplitN(this.RequestURI, "?", -1)
			this.requestPath = pieces[0]
			this.requestArgs = pieces[1]
		} else {
			this.requestPath = this.RequestURI
		}
	}

	// TODO init health check

	// headers
	if this.HeaderList != nil {
		this.hasRequestHeaders = len(this.HeaderList.RequestHeaders) > 0
	}
	this.hasResponseHeaders = len(this.ResponseHeaders) > 0

	// host
	this.hasHost = len(this.Host) > 0

	// variables
	// TODO 在host和port中支持变量
	this.hasAddrVariables = false

	return nil
}

// 候选对象代号
func (this *OriginServerConfig) CandidateCodes() []string {
	codes := []string{strconv.FormatInt(this.Id, 10)}
	if len(this.Code) > 0 {
		codes = append(codes, this.Code)
	}
	return codes
}

// 候选对象权重
func (this *OriginServerConfig) CandidateWeight() uint {
	return this.Weight
}

// 连接源站
func (this *OriginServerConfig) Connect() (net.Conn, error) {
	switch this.Scheme {
	case "", ProtocolTCP:
		// TODO 支持TCP4/TCP6
		// TODO 支持指定特定网卡
		// TODO Addr支持端口范围，如果有多个端口时，随机一个端口使用
		return net.DialTimeout("tcp", this.Addr.Host+":"+this.Addr.PortRange, this.failTimeoutDuration)
	case ProtocolTLS:
		// TODO 支持TCP4/TCP6
		// TODO 支持指定特定网卡
		// TODO Addr支持端口范围，如果有多个端口时，随机一个端口使用
		// TODO 支持使用证书
		return tls.Dial("tcp", this.Addr.Host+":"+this.Addr.PortRange, &tls.Config{})
	}

	// TODO 支持从Unix、Pipe、HTTP、HTTPS中读取数据

	return nil, errors.New("invalid scheme '" + this.Scheme + "'")
}
