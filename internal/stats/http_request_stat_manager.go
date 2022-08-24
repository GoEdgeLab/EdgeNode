package stats

import (
	iplib "github.com/TeaOSLab/EdgeCommon/pkg/iplibrary"
	"github.com/TeaOSLab/EdgeCommon/pkg/nodeconfigs"
	"github.com/TeaOSLab/EdgeCommon/pkg/rpc/pb"
	"github.com/TeaOSLab/EdgeNode/internal/events"
	"github.com/TeaOSLab/EdgeNode/internal/goman"
	"github.com/TeaOSLab/EdgeNode/internal/monitor"
	"github.com/TeaOSLab/EdgeNode/internal/remotelogs"
	"github.com/TeaOSLab/EdgeNode/internal/rpc"
	"github.com/TeaOSLab/EdgeNode/internal/trackers"
	"github.com/TeaOSLab/EdgeNode/internal/utils"
	"github.com/TeaOSLab/EdgeNode/internal/waf"
	"github.com/iwind/TeaGo/Tea"
	"github.com/iwind/TeaGo/maps"
	"github.com/iwind/TeaGo/types"
	timeutil "github.com/iwind/TeaGo/utils/time"
	"strconv"
	"strings"
	"time"
)

type StatItem struct {
	Bytes               int64
	CountRequests       int64
	CountAttackRequests int64
	AttackBytes         int64
}

var SharedHTTPRequestStatManager = NewHTTPRequestStatManager()

// HTTPRequestStatManager HTTP请求相关的统计
// 这里的统计是一个辅助统计，注意不要因为统计而影响服务工作性能
type HTTPRequestStatManager struct {
	ipChan                chan string
	userAgentChan         chan string
	firewallRuleGroupChan chan string

	cityMap     map[string]*StatItem // serverId@country@province@city => *StatItem ，不需要加锁，因为我们是使用channel依次执行的
	providerMap map[string]int64     // serverId@provider => count
	systemMap   map[string]int64     // serverId@system@version => count
	browserMap  map[string]int64     // serverId@browser@version => count

	dailyFirewallRuleGroupMap map[string]int64 // serverId@firewallRuleGroupId@action => count

	totalAttackRequests int64
}

// NewHTTPRequestStatManager 获取新对象
func NewHTTPRequestStatManager() *HTTPRequestStatManager {
	return &HTTPRequestStatManager{
		ipChan:                    make(chan string, 10_000), // TODO 将来可以配置容量
		userAgentChan:             make(chan string, 10_000), // TODO 将来可以配置容量
		firewallRuleGroupChan:     make(chan string, 10_000), // TODO 将来可以配置容量
		cityMap:                   map[string]*StatItem{},
		providerMap:               map[string]int64{},
		systemMap:                 map[string]int64{},
		browserMap:                map[string]int64{},
		dailyFirewallRuleGroupMap: map[string]int64{},
	}
}

// Start 启动
func (this *HTTPRequestStatManager) Start() {
	// 上传请求总数
	var monitorTicker = time.NewTicker(1 * time.Minute)
	events.OnKey(events.EventQuit, this, func() {
		monitorTicker.Stop()
	})
	goman.New(func() {
		for range monitorTicker.C {
			if this.totalAttackRequests > 0 {
				monitor.SharedValueQueue.Add(nodeconfigs.NodeValueItemAttackRequests, maps.Map{"total": this.totalAttackRequests})
				this.totalAttackRequests = 0
			}
		}
	})

	var loopTicker = time.NewTicker(1 * time.Second)
	var uploadTicker = time.NewTicker(30 * time.Minute)
	if Tea.IsTesting() {
		uploadTicker = time.NewTicker(10 * time.Second) // 在测试环境下缩短Ticker时间，以方便我们调试
	}
	remotelogs.Println("HTTP_REQUEST_STAT_MANAGER", "start ...")
	events.OnKey(events.EventQuit, this, func() {
		remotelogs.Println("HTTP_REQUEST_STAT_MANAGER", "quit")
		loopTicker.Stop()
		uploadTicker.Stop()
	})
	for range loopTicker.C {
		err := this.Loop()
		if err != nil {
			if rpc.IsConnError(err) {
				remotelogs.Warn("HTTP_REQUEST_STAT_MANAGER", err.Error())
			} else {
				remotelogs.Error("HTTP_REQUEST_STAT_MANAGER", err.Error())
			}
		}
		select {
		case <-uploadTicker.C:
			var tr = trackers.Begin("UPLOAD_REQUEST_STATS")
			err := this.Upload()
			tr.End()
			if err != nil {
				if !rpc.IsConnError(err) {
					remotelogs.Error("HTTP_REQUEST_STAT_MANAGER", "upload failed: "+err.Error())
				} else {
					remotelogs.Warn("HTTP_REQUEST_STAT_MANAGER", "upload failed: "+err.Error())
				}
			}
		default:

		}
	}
}

// AddRemoteAddr 添加客户端地址
func (this *HTTPRequestStatManager) AddRemoteAddr(serverId int64, remoteAddr string, bytes int64, isAttack bool) {
	if len(remoteAddr) == 0 {
		return
	}
	if remoteAddr[0] == '[' { // 排除IPv6
		return
	}
	var index = strings.Index(remoteAddr, ":")
	var ip string
	if index < 0 {
		ip = remoteAddr
	} else {
		ip = remoteAddr[:index]
	}
	if len(ip) > 0 {
		var s string
		if isAttack {
			s = strconv.FormatInt(serverId, 10) + "@" + ip + "@" + types.String(bytes) + "@1"
		} else {
			s = strconv.FormatInt(serverId, 10) + "@" + ip + "@" + types.String(bytes) + "@0"
		}
		select {
		case this.ipChan <- s:
		default:
			// 超出容量我们就丢弃
		}
	}
}

// AddUserAgent 添加UserAgent
func (this *HTTPRequestStatManager) AddUserAgent(serverId int64, userAgent string) {
	if len(userAgent) == 0 {
		return
	}

	select {
	case this.userAgentChan <- strconv.FormatInt(serverId, 10) + "@" + userAgent:
	default:
		// 超出容量我们就丢弃
	}
}

// AddFirewallRuleGroupId 添加防火墙拦截动作
func (this *HTTPRequestStatManager) AddFirewallRuleGroupId(serverId int64, firewallRuleGroupId int64, actions []*waf.ActionConfig) {
	if firewallRuleGroupId <= 0 {
		return
	}

	this.totalAttackRequests++

	for _, action := range actions {
		select {
		case this.firewallRuleGroupChan <- strconv.FormatInt(serverId, 10) + "@" + strconv.FormatInt(firewallRuleGroupId, 10) + "@" + action.Code:
		default:
			// 超出容量我们就丢弃
		}
	}
}

// Loop 单个循环
func (this *HTTPRequestStatManager) Loop() error {
	var timeout = time.NewTimer(10 * time.Minute) // 执行的最大时间
Loop:
	for {
		select {
		case ipString := <-this.ipChan:
			// serverId@ip@bytes@isAttack
			var pieces = strings.Split(ipString, "@")
			if len(pieces) < 4 {
				continue
			}
			var serverId = pieces[0]
			var ip = pieces[1]

			var result = iplib.LookupIP(ip)
			if result != nil && result.IsOk() {
				var key = serverId + "@" + result.CountryName() + "@" + result.ProvinceName() + "@" + result.CityName()
				stat, ok := this.cityMap[key]
				if !ok {
					stat = &StatItem{}
					this.cityMap[key] = stat
				}
				stat.Bytes += types.Int64(pieces[2])
				stat.CountRequests++
				if types.Int8(pieces[3]) == 1 {
					stat.AttackBytes += types.Int64(pieces[2])
					stat.CountAttackRequests++
				}

				if len(result.ProviderName()) > 0 {
					this.providerMap[serverId+"@"+result.ProviderName()]++
				}
			}

		case userAgentString := <-this.userAgentChan:
			var atIndex = strings.Index(userAgentString, "@")
			if atIndex < 0 {
				continue
			}
			var serverId = userAgentString[:atIndex]
			var userAgent = userAgentString[atIndex+1:]

			var result = SharedUserAgentParser.Parse(userAgent)
			var osInfo = result.OS
			if len(osInfo.Name) > 0 {
				dotIndex := strings.Index(osInfo.Version, ".")
				if dotIndex > -1 {
					osInfo.Version = osInfo.Version[:dotIndex]
				}
				this.systemMap[serverId+"@"+osInfo.Name+"@"+osInfo.Version]++
			}

			var browser, browserVersion = result.BrowserName, result.BrowserVersion
			if len(browser) > 0 {
				dotIndex := strings.Index(browserVersion, ".")
				if dotIndex > -1 {
					browserVersion = browserVersion[:dotIndex]
				}
				this.browserMap[serverId+"@"+browser+"@"+browserVersion]++
			}
		case firewallRuleGroupString := <-this.firewallRuleGroupChan:
			this.dailyFirewallRuleGroupMap[firewallRuleGroupString]++
		case <-timeout.C:
			break Loop
		default:
			break Loop
		}
	}

	timeout.Stop()

	return nil
}

// Upload 上传数据
func (this *HTTPRequestStatManager) Upload() error {
	// 上传统计数据
	rpcClient, err := rpc.SharedRPC()
	if err != nil {
		return err
	}

	// 月份相关
	var pbCities = []*pb.UploadServerHTTPRequestStatRequest_RegionCity{}
	var pbProviders = []*pb.UploadServerHTTPRequestStatRequest_RegionProvider{}
	var pbSystems = []*pb.UploadServerHTTPRequestStatRequest_System{}
	var pbBrowsers = []*pb.UploadServerHTTPRequestStatRequest_Browser{}
	for k, stat := range this.cityMap {
		var pieces = strings.SplitN(k, "@", 4)
		pbCities = append(pbCities, &pb.UploadServerHTTPRequestStatRequest_RegionCity{
			ServerId:            types.Int64(pieces[0]),
			CountryName:         pieces[1],
			ProvinceName:        pieces[2],
			CityName:            pieces[3],
			CountRequests:       stat.CountRequests,
			CountAttackRequests: stat.CountAttackRequests,
			Bytes:               stat.Bytes,
			AttackBytes:         stat.AttackBytes,
		})
	}
	for k, count := range this.providerMap {
		var pieces = strings.SplitN(k, "@", 2)
		pbProviders = append(pbProviders, &pb.UploadServerHTTPRequestStatRequest_RegionProvider{
			ServerId: types.Int64(pieces[0]),
			Name:     pieces[1],
			Count:    count,
		})
	}
	for k, count := range this.systemMap {
		var pieces = strings.SplitN(k, "@", 3)
		pbSystems = append(pbSystems, &pb.UploadServerHTTPRequestStatRequest_System{
			ServerId: types.Int64(pieces[0]),
			Name:     pieces[1],
			Version:  pieces[2],
			Count:    count,
		})
	}
	for k, count := range this.browserMap {
		var pieces = strings.SplitN(k, "@", 3)
		pbBrowsers = append(pbBrowsers, &pb.UploadServerHTTPRequestStatRequest_Browser{
			ServerId: types.Int64(pieces[0]),
			Name:     pieces[1],
			Version:  pieces[2],
			Count:    count,
		})
	}

	// 防火墙相关
	var pbFirewallRuleGroups = []*pb.UploadServerHTTPRequestStatRequest_HTTPFirewallRuleGroup{}
	for k, count := range this.dailyFirewallRuleGroupMap {
		var pieces = strings.SplitN(k, "@", 3)
		pbFirewallRuleGroups = append(pbFirewallRuleGroups, &pb.UploadServerHTTPRequestStatRequest_HTTPFirewallRuleGroup{
			ServerId:                types.Int64(pieces[0]),
			HttpFirewallRuleGroupId: types.Int64(pieces[1]),
			Action:                  pieces[2],
			Count:                   count,
		})
	}

	// 重置数据
	// 这里需要放到上传数据之前，防止因上传失败而导致统计数据堆积
	this.cityMap = map[string]*StatItem{}
	this.providerMap = map[string]int64{}
	this.systemMap = map[string]int64{}
	this.browserMap = map[string]int64{}
	this.dailyFirewallRuleGroupMap = map[string]int64{}

	// 上传数据
	_, err = rpcClient.ServerRPC.UploadServerHTTPRequestStat(rpcClient.Context(), &pb.UploadServerHTTPRequestStatRequest{
		Month:                  timeutil.Format("Ym"),
		Day:                    timeutil.Format("Ymd"),
		RegionCities:           pbCities,
		RegionProviders:        pbProviders,
		Systems:                pbSystems,
		Browsers:               pbBrowsers,
		HttpFirewallRuleGroups: pbFirewallRuleGroups,
	})
	if err != nil {
		// 是否包含了invalid UTF-8
		if strings.Contains(err.Error(), "string field contains invalid UTF-8") {
			for _, system := range pbSystems {
				system.Name = utils.ToValidUTF8string(system.Name)
				system.Version = utils.ToValidUTF8string(system.Version)
			}
			for _, browser := range pbBrowsers {
				browser.Name = utils.ToValidUTF8string(browser.Name)
				browser.Version = utils.ToValidUTF8string(browser.Version)
			}

			// 再次尝试
			_, err = rpcClient.ServerRPC.UploadServerHTTPRequestStat(rpcClient.Context(), &pb.UploadServerHTTPRequestStatRequest{
				Month:                  timeutil.Format("Ym"),
				Day:                    timeutil.Format("Ymd"),
				RegionCities:           pbCities,
				RegionProviders:        pbProviders,
				Systems:                pbSystems,
				Browsers:               pbBrowsers,
				HttpFirewallRuleGroups: pbFirewallRuleGroups,
			})
			if err != nil {
				return err
			}
		}
	}

	return nil
}
