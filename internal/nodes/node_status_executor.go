package nodes

import (
	"encoding/json"
	"github.com/TeaOSLab/EdgeCommon/pkg/nodeconfigs"
	"github.com/TeaOSLab/EdgeCommon/pkg/rpc/pb"
	"github.com/TeaOSLab/EdgeNode/internal/caches"
	teaconst "github.com/TeaOSLab/EdgeNode/internal/const"
	"github.com/TeaOSLab/EdgeNode/internal/events"
	"github.com/TeaOSLab/EdgeNode/internal/firewalls"
	"github.com/TeaOSLab/EdgeNode/internal/monitor"
	"github.com/TeaOSLab/EdgeNode/internal/remotelogs"
	"github.com/TeaOSLab/EdgeNode/internal/rpc"
	"github.com/TeaOSLab/EdgeNode/internal/trackers"
	"github.com/TeaOSLab/EdgeNode/internal/utils"
	fsutils "github.com/TeaOSLab/EdgeNode/internal/utils/fs"
	"github.com/iwind/TeaGo/lists"
	"github.com/iwind/TeaGo/maps"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/net"
	"math"
	"os"
	"runtime"
	"strings"
	"time"
)

type NodeStatusExecutor struct {
	isFirstTime     bool
	lastUpdatedTime time.Time

	cpuLogicalCount  int
	cpuPhysicalCount int

	// 流量统计
	lastIOCounterStat   net.IOCountersStat
	lastUDPInDatagrams  int64
	lastUDPOutDatagrams int64

	apiCallStat *rpc.CallStat

	ticker *time.Ticker
}

func NewNodeStatusExecutor() *NodeStatusExecutor {
	return &NodeStatusExecutor{
		ticker:      time.NewTicker(30 * time.Second),
		apiCallStat: rpc.NewCallStat(10),

		lastUDPInDatagrams:  -1,
		lastUDPOutDatagrams: -1,
	}
}

func (this *NodeStatusExecutor) Listen() {
	this.isFirstTime = true
	this.lastUpdatedTime = time.Now()
	this.update()

	events.OnKey(events.EventQuit, this, func() {
		remotelogs.Println("NODE_STATUS", "quit executor")
		this.ticker.Stop()
	})

	for range this.ticker.C {
		this.isFirstTime = false
		this.update()
	}
}

func (this *NodeStatusExecutor) update() {
	if sharedNodeConfig == nil {
		return
	}

	var tr = trackers.Begin("UPLOAD_NODE_STATUS")
	defer tr.End()

	var status = &nodeconfigs.NodeStatus{}
	status.BuildVersion = teaconst.Version
	status.BuildVersionCode = utils.VersionToLong(teaconst.Version)
	status.OS = runtime.GOOS
	status.Arch = runtime.GOARCH
	status.ExePath, _ = os.Executable()
	status.ConfigVersion = sharedNodeConfig.Version
	status.IsActive = true
	status.ConnectionCount = sharedListenerManager.TotalActiveConnections()
	status.CacheTotalDiskSize = caches.SharedManager.TotalDiskSize()
	status.CacheTotalMemorySize = caches.SharedManager.TotalMemorySize()
	status.TrafficInBytes = teaconst.InTrafficBytes
	status.TrafficOutBytes = teaconst.OutTrafficBytes

	apiSuccessPercent, apiAvgCostSeconds := this.apiCallStat.Sum()
	status.APISuccessPercent = apiSuccessPercent
	status.APIAvgCostSeconds = apiAvgCostSeconds

	var localFirewall = firewalls.Firewall()
	if localFirewall != nil && !localFirewall.IsMock() {
		status.LocalFirewallName = localFirewall.Name()
	}

	// 记录监控数据
	monitor.SharedValueQueue.Add(nodeconfigs.NodeValueItemConnections, maps.Map{
		"total": status.ConnectionCount,
	})

	hostname, _ := os.Hostname()
	status.Hostname = hostname

	var cpuTR = tr.Begin("cpu")
	this.updateCPU(status)
	cpuTR.End()

	var memTR = tr.Begin("memory")
	this.updateMem(status)
	memTR.End()

	var loadTR = tr.Begin("load")
	this.updateLoad(status)
	loadTR.End()

	var diskTR = tr.Begin("disk")
	this.updateDisk(status)
	diskTR.End()

	var cacheSpaceTR = tr.Begin("cache space")
	this.updateCacheSpace(status)
	cacheSpaceTR.End()

	this.updateAllTraffic(status)

	// 修改更新时间
	this.lastUpdatedTime = time.Now()

	status.UpdatedAt = time.Now().Unix()
	status.Timestamp = status.UpdatedAt

	//  发送数据
	jsonData, err := json.Marshal(status)
	if err != nil {
		remotelogs.Error("NODE_STATUS", "serial NodeStatus fail: "+err.Error())
		return
	}
	rpcClient, err := rpc.SharedRPC()
	if err != nil {
		remotelogs.Error("NODE_STATUS", "failed to open rpc: "+err.Error())
		return
	}

	var before = time.Now()
	_, err = rpcClient.NodeRPC.UpdateNodeStatus(rpcClient.Context(), &pb.UpdateNodeStatusRequest{
		StatusJSON: jsonData,
	})
	var costSeconds = time.Since(before).Seconds()
	this.apiCallStat.Add(err == nil, costSeconds)
	if err != nil {
		if rpc.IsConnError(err) {
			remotelogs.Warn("NODE_STATUS", "rpc UpdateNodeStatus() failed: "+err.Error())
		} else {
			remotelogs.Error("NODE_STATUS", "rpc UpdateNodeStatus() failed: "+err.Error())
		}
		return
	}
}

// 更新CPU
func (this *NodeStatusExecutor) updateCPU(status *nodeconfigs.NodeStatus) {
	var duration = time.Duration(0)
	if this.isFirstTime {
		duration = 100 * time.Millisecond
	}
	percents, err := cpu.Percent(duration, false)
	if err != nil {
		status.Error = "cpu.Percent(): " + err.Error()
		return
	}
	if len(percents) == 0 {
		return
	}
	status.CPUUsage = percents[0] / 100

	// 记录监控数据
	monitor.SharedValueQueue.Add(nodeconfigs.NodeValueItemCPU, maps.Map{
		"usage": status.CPUUsage,
		"cores": runtime.NumCPU(),
	})

	if this.cpuLogicalCount == 0 && this.cpuPhysicalCount == 0 {
		status.CPULogicalCount, err = cpu.Counts(true)
		if err != nil {
			status.Error = "cpu.Counts(): " + err.Error()
			return
		}
		status.CPUPhysicalCount, err = cpu.Counts(false)
		if err != nil {
			status.Error = "cpu.Counts(): " + err.Error()
			return
		}
		this.cpuLogicalCount = status.CPULogicalCount
		this.cpuPhysicalCount = status.CPUPhysicalCount
	} else {
		status.CPULogicalCount = this.cpuLogicalCount
		status.CPUPhysicalCount = this.cpuPhysicalCount
	}
}

// 更新硬盘
func (this *NodeStatusExecutor) updateDisk(status *nodeconfigs.NodeStatus) {
	status.DiskWritingSpeedMB = int(fsutils.DiskSpeedMB)

	partitions, err := disk.Partitions(false)
	if err != nil {
		remotelogs.Error("NODE_STATUS", err.Error())
		return
	}
	lists.Sort(partitions, func(i int, j int) bool {
		p1 := partitions[i]
		p2 := partitions[j]
		return p1.Mountpoint > p2.Mountpoint
	})

	// 当前TeaWeb所在的fs
	var rootFS = ""
	var rootTotal = uint64(0)
	var totalUsed = uint64(0)
	if lists.ContainsString([]string{"darwin", "linux", "freebsd"}, runtime.GOOS) {
		for _, p := range partitions {
			if p.Mountpoint == "/" {
				rootFS = p.Fstype
				usage, _ := disk.Usage(p.Mountpoint)
				if usage != nil {
					rootTotal = usage.Total
					totalUsed = usage.Used
				}
				break
			}
		}
	}

	var total = rootTotal
	var maxUsage = float64(0)
	for _, partition := range partitions {
		if runtime.GOOS != "windows" && !strings.Contains(partition.Device, "/") && !strings.Contains(partition.Device, "\\") {
			continue
		}

		// 跳过不同fs的
		if len(rootFS) > 0 && rootFS != partition.Fstype {
			continue
		}

		usage, err := disk.Usage(partition.Mountpoint)
		if err != nil {
			continue
		}

		if partition.Mountpoint != "/" && (usage.Total != rootTotal || total == 0) {
			total += usage.Total
			totalUsed += usage.Used
			if usage.UsedPercent >= maxUsage {
				maxUsage = usage.UsedPercent
				status.DiskMaxUsagePartition = partition.Mountpoint
			}
		}
	}
	status.DiskTotal = total
	if total > 0 {
		status.DiskUsage = float64(totalUsed) / float64(total)
	}
	status.DiskMaxUsage = maxUsage / 100

	// 记录监控数据
	monitor.SharedValueQueue.Add(nodeconfigs.NodeValueItemDisk, maps.Map{
		"total":    status.DiskTotal,
		"usage":    status.DiskUsage,
		"maxUsage": status.DiskMaxUsage,
	})
}

// 缓存空间
func (this *NodeStatusExecutor) updateCacheSpace(status *nodeconfigs.NodeStatus) {
	var result = []maps.Map{}
	var cachePaths = caches.SharedManager.FindAllCachePaths()
	for _, path := range cachePaths {
		stat, err := fsutils.StatDevice(path)
		if err != nil {
			return
		}
		result = append(result, maps.Map{
			"path":  path,
			"total": stat.TotalSize(),
			"avail": stat.FreeSize(),
			"used":  stat.UsedSize(),
		})
	}
	monitor.SharedValueQueue.Add(nodeconfigs.NodeValueItemCacheDir, maps.Map{
		"dirs": result,
	})
}

// 流量
func (this *NodeStatusExecutor) updateAllTraffic(status *nodeconfigs.NodeStatus) {
	trafficCounters, err := net.IOCounters(true)
	if err != nil {
		remotelogs.Warn("NODE_STATUS_EXECUTOR", err.Error())
		return
	}

	var allCounter = net.IOCountersStat{}
	for _, counter := range trafficCounters {
		// 跳过lo
		if counter.Name == "lo" {
			continue
		}
		allCounter.BytesRecv += counter.BytesRecv
		allCounter.BytesSent += counter.BytesSent
	}
	if allCounter.BytesSent == 0 && allCounter.BytesRecv == 0 {
		return
	}

	if this.lastIOCounterStat.BytesSent > 0 {
		// 记录监控数据
		if allCounter.BytesSent >= this.lastIOCounterStat.BytesSent && allCounter.BytesRecv >= this.lastIOCounterStat.BytesRecv {
			var costSeconds = int(math.Ceil(time.Since(this.lastUpdatedTime).Seconds()))
			if costSeconds > 0 {
				var bytesSent = allCounter.BytesSent - this.lastIOCounterStat.BytesSent
				var bytesRecv = allCounter.BytesRecv - this.lastIOCounterStat.BytesRecv

				// UDP
				var udpInDatagrams int64 = 0
				var udpOutDatagrams int64 = 0
				protoStats, protoErr := net.ProtoCounters([]string{"udp"})
				if protoErr == nil {
					for _, protoStat := range protoStats {
						if protoStat.Protocol == "udp" {
							udpInDatagrams = protoStat.Stats["InDatagrams"]
							udpOutDatagrams = protoStat.Stats["OutDatagrams"]
							if udpInDatagrams < 0 {
								udpInDatagrams = 0
							}
							if udpOutDatagrams < 0 {
								udpOutDatagrams = 0
							}
						}
					}
				}

				var avgUDPInDatagrams int64 = 0
				var avgUDPOutDatagrams int64 = 0
				if this.lastUDPInDatagrams >= 0 && this.lastUDPOutDatagrams >= 0 {
					avgUDPInDatagrams = (udpInDatagrams - this.lastUDPInDatagrams) / int64(costSeconds)
					avgUDPOutDatagrams = (udpOutDatagrams - this.lastUDPOutDatagrams) / int64(costSeconds)
					if avgUDPInDatagrams < 0 {
						avgUDPInDatagrams = 0
					}
					if avgUDPOutDatagrams < 0 {
						avgUDPOutDatagrams = 0
					}
				}

				this.lastUDPInDatagrams = udpInDatagrams
				this.lastUDPOutDatagrams = udpOutDatagrams

				monitor.SharedValueQueue.Add(nodeconfigs.NodeValueItemAllTraffic, maps.Map{
					"inBytes":     bytesRecv,
					"outBytes":    bytesSent,
					"avgInBytes":  bytesRecv / uint64(costSeconds),
					"avgOutBytes": bytesSent / uint64(costSeconds),

					"avgUDPInDatagrams":  avgUDPInDatagrams,
					"avgUDPOutDatagrams": avgUDPOutDatagrams,
				})
			}
		}
	}
	this.lastIOCounterStat = allCounter
}
