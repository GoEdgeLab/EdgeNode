package stats

import (
	"github.com/TeaOSLab/EdgeCommon/pkg/rpc/pb"
	"github.com/TeaOSLab/EdgeNode/internal/events"
	"github.com/TeaOSLab/EdgeNode/internal/iplibrary"
	"github.com/TeaOSLab/EdgeNode/internal/remotelogs"
	"github.com/TeaOSLab/EdgeNode/internal/rpc"
	"github.com/iwind/TeaGo/Tea"
	"github.com/iwind/TeaGo/types"
	timeutil "github.com/iwind/TeaGo/utils/time"
	"github.com/mssola/user_agent"
	"strconv"
	"strings"
	"time"
)

var SharedHTTPRequestStatManager = NewHTTPRequestStatManager()

// HTTP请求相关的统计
// 这里的统计是一个辅助统计，注意不要因为统计而影响服务工作性能
type HTTPRequestStatManager struct {
	ipChan                chan string
	userAgentChan         chan string
	firewallRuleGroupChan chan string

	cityMap     map[string]int64 // serverId@country@province@city => count ，不需要加锁，因为我们是使用channel依次执行的
	providerMap map[string]int64 // serverId@provider => count
	systemMap   map[string]int64 // serverId@system@version => count
	browserMap  map[string]int64 // serverId@browser@version => count

	dailyFirewallRuleGroupMap map[string]int64 // serverId@firewallRuleGroupId@action => count
}

// 获取新对象
func NewHTTPRequestStatManager() *HTTPRequestStatManager {
	return &HTTPRequestStatManager{
		ipChan:                    make(chan string, 10_000), // TODO 将来可以配置容量
		userAgentChan:             make(chan string, 10_000), // TODO 将来可以配置容量
		firewallRuleGroupChan:     make(chan string, 10_000), // TODO 将来可以配置容量
		cityMap:            map[string]int64{},
		providerMap:        map[string]int64{},
		systemMap:          map[string]int64{},
		browserMap:         map[string]int64{},
		dailyFirewallRuleGroupMap: map[string]int64{},
	}
}

// 启动
func (this *HTTPRequestStatManager) Start() {
	loopTicker := time.NewTicker(1 * time.Second)
	uploadTicker := time.NewTicker(30 * time.Minute)
	if Tea.IsTesting() {
		uploadTicker = time.NewTicker(10 * time.Second) // 在测试环境下缩短Ticker时间，以方便我们调试
	}
	remotelogs.Println("HTTP_REQUEST_STAT_MANAGER", "start ...")
	events.On(events.EventQuit, func() {
		remotelogs.Println("HTTP_REQUEST_STAT_MANAGER", "quit")
		loopTicker.Stop()
		uploadTicker.Stop()
	})
	for range loopTicker.C {
		err := this.Loop()
		if err != nil {
			remotelogs.Error("HTTP_REQUEST_STAT_MANAGER", err.Error())
		}
		select {
		case <-uploadTicker.C:
			err := this.Upload()
			if err != nil {
				remotelogs.Error("HTTP_REQUEST_STAT_MANAGER", "upload failed: "+err.Error())
			}
		default:

		}
	}
}

// 添加客户端地址
func (this *HTTPRequestStatManager) AddRemoteAddr(serverId int64, remoteAddr string) {
	if len(remoteAddr) == 0 {
		return
	}
	if remoteAddr[0] == '[' { // 排除IPv6
		return
	}
	index := strings.Index(remoteAddr, ":")
	var ip string
	if index < 0 {
		ip = remoteAddr
	} else {
		ip = remoteAddr[:index]
	}
	if len(ip) > 0 {
		select {
		case this.ipChan <- strconv.FormatInt(serverId, 10) + "@" + ip:
		default:
			// 超出容量我们就丢弃
		}
	}
}

// 添加UserAgent
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

// 添加防火墙拦截动作
func (this *HTTPRequestStatManager) AddFirewallRuleGroupId(serverId int64, firewallRuleGroupId int64, action string) {
	if firewallRuleGroupId <= 0 {
		return
	}
	select {
	case this.firewallRuleGroupChan <- strconv.FormatInt(serverId, 10) + "@" + strconv.FormatInt(firewallRuleGroupId, 10) + "@" + action:
	default:
		// 超出容量我们就丢弃
	}
}

// 单个循环
func (this *HTTPRequestStatManager) Loop() error {
	timeout := time.NewTimer(10 * time.Minute) // 执行的最大时间
	userAgentParser := &user_agent.UserAgent{}
Loop:
	for {
		select {
		case ipString := <-this.ipChan:
			atIndex := strings.Index(ipString, "@")
			if atIndex < 0 {
				continue
			}
			serverId := ipString[:atIndex]
			ip := ipString[atIndex+1:]
			if iplibrary.SharedLibrary != nil {
				result, err := iplibrary.SharedLibrary.Lookup(ip)
				if err == nil {
					this.cityMap[serverId+"@"+result.Country+"@"+result.Province+"@"+result.City]  ++

					if len(result.ISP) > 0 {
						this.providerMap[serverId+"@"+result.ISP] ++
					}
				}
			}
		case userAgentString := <-this.userAgentChan:
			atIndex := strings.Index(userAgentString, "@")
			if atIndex < 0 {
				continue
			}
			serverId := userAgentString[:atIndex]
			userAgent := userAgentString[atIndex+1:]

			userAgentParser.Parse(userAgent)
			osInfo := userAgentParser.OSInfo()
			if len(osInfo.Name) > 0 {
				dotIndex := strings.Index(osInfo.Version, ".")
				if dotIndex > -1 {
					osInfo.Version = osInfo.Version[:dotIndex]
				}
				this.systemMap[serverId+"@"+osInfo.Name+"@"+osInfo.Version]++
			}

			browser, browserVersion := userAgentParser.Browser()
			if len(browser) > 0 {
				dotIndex := strings.Index(browserVersion, ".")
				if dotIndex > -1 {
					browserVersion = browserVersion[:dotIndex]
				}
				this.browserMap[serverId+"@"+browser+"@"+browserVersion] ++
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

func (this *HTTPRequestStatManager) Upload() error {
	// 上传统计数据
	rpcClient, err := rpc.SharedRPC()
	if err != nil {
		return err
	}

	// 月份相关
	pbCities := []*pb.UploadServerHTTPRequestStatRequest_RegionCity{}
	pbProviders := []*pb.UploadServerHTTPRequestStatRequest_RegionProvider{}
	pbSystems := []*pb.UploadServerHTTPRequestStatRequest_System{}
	pbBrowsers := []*pb.UploadServerHTTPRequestStatRequest_Browser{}
	for k, count := range this.cityMap {
		pieces := strings.SplitN(k, "@", 4)
		pbCities = append(pbCities, &pb.UploadServerHTTPRequestStatRequest_RegionCity{
			ServerId:     types.Int64(pieces[0]),
			CountryName:  pieces[1],
			ProvinceName: pieces[2],
			CityName:     pieces[3],
			Count:        count,
		})
	}
	for k, count := range this.providerMap {
		pieces := strings.SplitN(k, "@", 2)
		pbProviders = append(pbProviders, &pb.UploadServerHTTPRequestStatRequest_RegionProvider{
			ServerId: types.Int64(pieces[0]),
			Name:     pieces[1],
			Count:    count,
		})
	}
	for k, count := range this.systemMap {
		pieces := strings.SplitN(k, "@", 3)
		pbSystems = append(pbSystems, &pb.UploadServerHTTPRequestStatRequest_System{
			ServerId: types.Int64(pieces[0]),
			Name:     pieces[1],
			Version:  pieces[2],
			Count:    count,
		})
	}
	for k, count := range this.browserMap {
		pieces := strings.SplitN(k, "@", 3)
		pbBrowsers = append(pbBrowsers, &pb.UploadServerHTTPRequestStatRequest_Browser{
			ServerId: types.Int64(pieces[0]),
			Name:     pieces[1],
			Version:  pieces[2],
			Count:    count,
		})
	}

	// 防火墙相关
	pbFirewallRuleGroups := []*pb.UploadServerHTTPRequestStatRequest_HTTPFirewallRuleGroup{}
	for k, count := range this.dailyFirewallRuleGroupMap {
		pieces := strings.SplitN(k, "@", 3)
		pbFirewallRuleGroups = append(pbFirewallRuleGroups, &pb.UploadServerHTTPRequestStatRequest_HTTPFirewallRuleGroup{
			ServerId:                types.Int64(pieces[0]),
			HttpFirewallRuleGroupId: types.Int64(pieces[1]),
			Action:                  pieces[2],
			Count:                   count,
		})
	}

	_, err = rpcClient.ServerRPC().UploadServerHTTPRequestStat(rpcClient.Context(), &pb.UploadServerHTTPRequestStatRequest{
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

	// 重置数据
	this.cityMap = map[string]int64{}
	this.providerMap = map[string]int64{}
	this.systemMap = map[string]int64{}
	this.browserMap = map[string]int64{}
	this.dailyFirewallRuleGroupMap = map[string]int64{}
	return nil
}
