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
	"github.com/TeaOSLab/EdgeNode/internal/utils/agents"
	"github.com/TeaOSLab/EdgeNode/internal/waf"
	"github.com/iwind/TeaGo/Tea"
	"github.com/iwind/TeaGo/maps"
	"github.com/iwind/TeaGo/types"
	timeutil "github.com/iwind/TeaGo/utils/time"
	"sort"
	"strconv"
	"strings"
	"sync"
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

	serverCityCountMap    map[string]int16 // serverIdString => count cities
	serverSystemCountMap  map[string]int16 // serverIdString => count systems
	serverBrowserCountMap map[string]int16 // serverIdString => count browsers

	totalAttackRequests int64

	locker sync.Mutex

	monitorTicker *time.Ticker
	uploadTicker  *time.Ticker
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

		serverCityCountMap:    map[string]int16{},
		serverSystemCountMap:  map[string]int16{},
		serverBrowserCountMap: map[string]int16{},
	}
}

// Start 启动
func (this *HTTPRequestStatManager) Start() {
	// 上传请求总数
	this.monitorTicker = time.NewTicker(1 * time.Minute)
	events.OnKey(events.EventQuit, this, func() {
		this.monitorTicker.Stop()
	})
	goman.New(func() {
		for range this.monitorTicker.C {
			if this.totalAttackRequests > 0 {
				monitor.SharedValueQueue.Add(nodeconfigs.NodeValueItemAttackRequests, maps.Map{"total": this.totalAttackRequests})
				this.totalAttackRequests = 0
			}
		}
	})

	this.uploadTicker = time.NewTicker(30 * time.Minute)
	if Tea.IsTesting() {
		this.uploadTicker = time.NewTicker(10 * time.Second) // 在测试环境下缩短Ticker时间，以方便我们调试
	}
	remotelogs.Println("HTTP_REQUEST_STAT_MANAGER", "start ...")
	events.OnKey(events.EventQuit, this, func() {
		remotelogs.Println("HTTP_REQUEST_STAT_MANAGER", "quit")
		this.uploadTicker.Stop()
	})

	// 上传Ticker
	goman.New(func() {
		for range this.uploadTicker.C {
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

		}
	})

	// 分析Ticker
	for {
		err := this.Loop()
		if err != nil {
			if rpc.IsConnError(err) {
				remotelogs.Warn("HTTP_REQUEST_STAT_MANAGER", err.Error())
			} else {
				remotelogs.Error("HTTP_REQUEST_STAT_MANAGER", err.Error())
			}
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
func (this *HTTPRequestStatManager) AddUserAgent(serverId int64, userAgent string, ip string) {
	if len(userAgent) == 0 || strings.ContainsRune(userAgent, '@') /** 非常重要，防止后面组合字符串时出现异常 **/ {
		return
	}

	// 是否包含一些知名Agent
	if len(userAgent) > 0 && len(ip) > 0 && agents.IsAgentFromUserAgent(userAgent) {
		agents.SharedQueue.Push(ip)
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
	select {
	case ipString := <-this.ipChan:
		// serverId@ip@bytes@isAttack
		var pieces = strings.Split(ipString, "@")
		if len(pieces) < 4 {
			return nil
		}
		var serverIdString = pieces[0]
		var ip = pieces[1]

		var result = iplib.LookupIP(ip)
		if result != nil && result.IsOk() {
			this.locker.Lock()
			if result.CountryId() > 0 {
				var key = serverIdString + "@" + types.String(result.CountryId()) + "@" + types.String(result.ProvinceId()) + "@" + types.String(result.CityId())
				stat, ok := this.cityMap[key]
				if !ok {
					// 检查数量
					if this.serverCityCountMap[serverIdString] > 128 { // 限制单个服务的城市数量，防止数量过多
						this.locker.Unlock()
						return nil
					}
					this.serverCityCountMap[serverIdString]++ // 需要放在限制之后，因为使用的是int16

					stat = &StatItem{}
					this.cityMap[key] = stat
				}
				stat.Bytes += types.Int64(pieces[2])
				stat.CountRequests++
				if types.Int8(pieces[3]) == 1 {
					stat.AttackBytes += types.Int64(pieces[2])
					stat.CountAttackRequests++
				}
			}

			if result.ProviderId() > 0 {
				this.providerMap[serverIdString+"@"+types.String(result.ProviderId())]++
			} else if utils.IsLocalIP(ip) { // 局域网IP
				this.providerMap[serverIdString+"@258"]++
			}
			this.locker.Unlock()
		}
	case userAgentString := <-this.userAgentChan:
		var atIndex = strings.Index(userAgentString, "@")
		if atIndex < 0 {
			return nil
		}
		var serverIdString = userAgentString[:atIndex]
		var userAgent = userAgentString[atIndex+1:]

		var result = SharedUserAgentParser.Parse(userAgent)
		var osInfo = result.OS
		if len(osInfo.Name) > 0 {
			dotIndex := strings.Index(osInfo.Version, ".")
			if dotIndex > -1 {
				osInfo.Version = osInfo.Version[:dotIndex]
			}
			this.locker.Lock()

			var systemKey = serverIdString + "@" + osInfo.Name + "@" + osInfo.Version
			_, ok := this.systemMap[systemKey]
			if !ok {
				if this.serverSystemCountMap[serverIdString] < 128 { // 限制最大数据，防止攻击
					this.serverSystemCountMap[serverIdString]++
					ok = true
				}
			}
			if ok {
				this.systemMap[systemKey]++
			}
			this.locker.Unlock()
		}

		var browser, browserVersion = result.BrowserName, result.BrowserVersion
		if len(browser) > 0 {
			dotIndex := strings.Index(browserVersion, ".")
			if dotIndex > -1 {
				browserVersion = browserVersion[:dotIndex]
			}
			this.locker.Lock()

			var browserKey = serverIdString + "@" + browser + "@" + browserVersion
			_, ok := this.browserMap[browserKey]
			if !ok {
				if this.serverBrowserCountMap[serverIdString] < 256 { // 限制最大数据，防止攻击
					this.serverBrowserCountMap[serverIdString]++
					ok = true
				}
			}
			if ok {
				this.browserMap[browserKey]++
			}
			this.locker.Unlock()
		}
	case firewallRuleGroupString := <-this.firewallRuleGroupChan:
		this.locker.Lock()
		this.dailyFirewallRuleGroupMap[firewallRuleGroupString]++
		this.locker.Unlock()
	}

	return nil
}

// Upload 上传数据
func (this *HTTPRequestStatManager) Upload() error {
	// 上传统计数据
	rpcClient, err := rpc.SharedRPC()
	if err != nil {
		return err
	}

	// 拷贝数据
	this.locker.Lock()
	var cityMap = this.cityMap
	var providerMap = this.providerMap
	var systemMap = this.systemMap
	var browserMap = this.browserMap
	var dailyFirewallRuleGroupMap = this.dailyFirewallRuleGroupMap

	this.cityMap = map[string]*StatItem{}
	this.providerMap = map[string]int64{}
	this.systemMap = map[string]int64{}
	this.browserMap = map[string]int64{}
	this.dailyFirewallRuleGroupMap = map[string]int64{}

	this.serverCityCountMap = map[string]int16{}
	this.serverSystemCountMap = map[string]int16{}
	this.serverBrowserCountMap = map[string]int16{}

	this.locker.Unlock()

	// 上传限制
	var maxCities int16 = 32
	var maxProviders int16 = 32
	var maxSystems int16 = 64
	var maxBrowsers int16 = 64
	nodeConfig, _ := nodeconfigs.SharedNodeConfig()
	if nodeConfig != nil {
		var serverConfig = nodeConfig.GlobalServerConfig // 复制是为了防止在中途修改
		if serverConfig != nil {
			var uploadConfig = serverConfig.Stat.Upload
			if uploadConfig.MaxCities > 0 {
				maxCities = uploadConfig.MaxCities
			}
			if uploadConfig.MaxProviders > 0 {
				maxProviders = uploadConfig.MaxProviders
			}
			if uploadConfig.MaxSystems > 0 {
				maxSystems = uploadConfig.MaxSystems
			}
			if uploadConfig.MaxBrowsers > 0 {
				maxBrowsers = uploadConfig.MaxBrowsers
			}
		}
	}

	var pbCities = []*pb.UploadServerHTTPRequestStatRequest_RegionCity{}
	var pbProviders = []*pb.UploadServerHTTPRequestStatRequest_RegionProvider{}
	var pbSystems = []*pb.UploadServerHTTPRequestStatRequest_System{}
	var pbBrowsers = []*pb.UploadServerHTTPRequestStatRequest_Browser{}

	// 城市
	for k, stat := range cityMap {
		var pieces = strings.SplitN(k, "@", 4)
		var serverId = types.Int64(pieces[0])
		pbCities = append(pbCities, &pb.UploadServerHTTPRequestStatRequest_RegionCity{
			ServerId:            serverId,
			CountryId:           types.Int64(pieces[1]),
			ProvinceId:          types.Int64(pieces[2]),
			CityId:              types.Int64(pieces[3]),
			CountRequests:       stat.CountRequests,
			CountAttackRequests: stat.CountAttackRequests,
			Bytes:               stat.Bytes,
			AttackBytes:         stat.AttackBytes,
		})
	}
	if len(cityMap) > int(maxCities) {
		var newPBCities = []*pb.UploadServerHTTPRequestStatRequest_RegionCity{}
		sort.Slice(pbCities, func(i, j int) bool {
			return pbCities[i].CountRequests > pbCities[j].CountRequests
		})
		var serverCountMap = map[int64]int16{} // serverId => count
		for _, city := range pbCities {
			serverCountMap[city.ServerId]++
			if serverCountMap[city.ServerId] > maxCities {
				continue
			}
			newPBCities = append(newPBCities, city)
		}
		if len(pbCities) != len(newPBCities) {
			pbCities = newPBCities
		}
	}

	// 运营商
	for k, count := range providerMap {
		var pieces = strings.SplitN(k, "@", 2)
		var serverId = types.Int64(pieces[0])
		pbProviders = append(pbProviders, &pb.UploadServerHTTPRequestStatRequest_RegionProvider{
			ServerId:   serverId,
			ProviderId: types.Int64(pieces[1]),
			Count:      count,
		})
	}
	if len(providerMap) > int(maxProviders) {
		var newPBProviders = []*pb.UploadServerHTTPRequestStatRequest_RegionProvider{}
		sort.Slice(pbProviders, func(i, j int) bool {
			return pbProviders[i].Count > pbProviders[j].Count
		})
		var serverCountMap = map[int64]int16{}
		for _, provider := range pbProviders {
			serverCountMap[provider.ServerId]++
			if serverCountMap[provider.ServerId] > maxProviders {
				continue
			}
			newPBProviders = append(newPBProviders, provider)
		}
		if len(pbProviders) != len(newPBProviders) {
			pbProviders = newPBProviders
		}
	}

	// 操作系统
	for k, count := range systemMap {
		var pieces = strings.SplitN(k, "@", 3)
		var serverId = types.Int64(pieces[0])
		pbSystems = append(pbSystems, &pb.UploadServerHTTPRequestStatRequest_System{
			ServerId: serverId,
			Name:     pieces[1],
			Version:  pieces[2],
			Count:    count,
		})
	}
	if len(systemMap) > int(maxSystems) {
		var newPBSystems = []*pb.UploadServerHTTPRequestStatRequest_System{}
		sort.Slice(pbSystems, func(i, j int) bool {
			return pbSystems[i].Count > pbSystems[j].Count
		})
		var serverCountMap = map[int64]int16{}
		for _, system := range pbSystems {
			serverCountMap[system.ServerId]++
			if serverCountMap[system.ServerId] > maxSystems {
				continue
			}
			newPBSystems = append(newPBSystems, system)
		}
		if len(pbSystems) != len(newPBSystems) {
			pbSystems = newPBSystems
		}
	}

	// 浏览器
	for k, count := range browserMap {
		var pieces = strings.SplitN(k, "@", 3)
		var serverId = types.Int64(pieces[0])
		pbBrowsers = append(pbBrowsers, &pb.UploadServerHTTPRequestStatRequest_Browser{
			ServerId: serverId,
			Name:     pieces[1],
			Version:  pieces[2],
			Count:    count,
		})
	}
	if len(browserMap) > int(maxBrowsers) {
		var newPBBrowsers = []*pb.UploadServerHTTPRequestStatRequest_Browser{}
		sort.Slice(pbBrowsers, func(i, j int) bool {
			return pbBrowsers[i].Count > pbBrowsers[j].Count
		})
		var serverCountMap = map[int64]int16{}
		for _, browser := range pbBrowsers {
			serverCountMap[browser.ServerId]++
			if serverCountMap[browser.ServerId] > maxBrowsers {
				continue
			}
			newPBBrowsers = append(newPBBrowsers, browser)
		}
		if len(pbBrowsers) != len(newPBBrowsers) {
			pbBrowsers = newPBBrowsers
		}
	}

	// 防火墙相关
	var pbFirewallRuleGroups = []*pb.UploadServerHTTPRequestStatRequest_HTTPFirewallRuleGroup{}
	for k, count := range dailyFirewallRuleGroupMap {
		var pieces = strings.SplitN(k, "@", 3)
		pbFirewallRuleGroups = append(pbFirewallRuleGroups, &pb.UploadServerHTTPRequestStatRequest_HTTPFirewallRuleGroup{
			ServerId:                types.Int64(pieces[0]),
			HttpFirewallRuleGroupId: types.Int64(pieces[1]),
			Action:                  pieces[2],
			Count:                   count,
		})
	}

	// 检查是否有数据
	if len(pbCities) == 0 &&
		len(pbProviders) == 0 &&
		len(pbSystems) == 0 &&
		len(pbBrowsers) == 0 &&
		len(pbFirewallRuleGroups) == 0 {
		return nil
	}

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
