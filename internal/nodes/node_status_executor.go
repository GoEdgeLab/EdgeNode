package nodes

import (
	"encoding/json"
	"github.com/TeaOSLab/EdgeCommon/pkg/nodeconfigs"
	"github.com/TeaOSLab/EdgeCommon/pkg/rpc/pb"
	"github.com/TeaOSLab/EdgeNode/internal/caches"
	teaconst "github.com/TeaOSLab/EdgeNode/internal/const"
	"github.com/TeaOSLab/EdgeNode/internal/events"
	"github.com/TeaOSLab/EdgeNode/internal/monitor"
	"github.com/TeaOSLab/EdgeNode/internal/remotelogs"
	"github.com/TeaOSLab/EdgeNode/internal/rpc"
	"github.com/TeaOSLab/EdgeNode/internal/utils"
	"github.com/iwind/TeaGo/lists"
	"github.com/iwind/TeaGo/maps"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/disk"
	"os"
	"runtime"
	"strings"
	"time"
)

type NodeStatusExecutor struct {
	isFirstTime bool

	cpuUpdatedTime   time.Time
	cpuLogicalCount  int
	cpuPhysicalCount int
}

func NewNodeStatusExecutor() *NodeStatusExecutor {
	return &NodeStatusExecutor{}
}

func (this *NodeStatusExecutor) Listen() {
	this.isFirstTime = true
	this.cpuUpdatedTime = time.Now()
	this.update()

	// TODO 这个时间间隔可以配置
	ticker := time.NewTicker(30 * time.Second)

	events.On(events.EventQuit, func() {
		remotelogs.Println("NODE_STATUS", "quit executor")
		ticker.Stop()
	})

	for range ticker.C {
		this.isFirstTime = false
		this.update()
	}
}

func (this *NodeStatusExecutor) update() {
	if sharedNodeConfig == nil {
		return
	}

	status := &nodeconfigs.NodeStatus{}
	status.BuildVersion = teaconst.Version
	status.BuildVersionCode = utils.VersionToLong(teaconst.Version)
	status.OS = runtime.GOOS
	status.Arch = runtime.GOARCH
	status.ConfigVersion = sharedNodeConfig.Version
	status.IsActive = true
	status.ConnectionCount = sharedListenerManager.TotalActiveConnections()
	status.CacheTotalDiskSize = caches.SharedManager.TotalDiskSize()
	status.CacheTotalMemorySize = caches.SharedManager.TotalMemorySize()

	// 记录监控数据
	monitor.SharedValueQueue.Add(nodeconfigs.NodeValueItemConnections, maps.Map{
		"total": status.ConnectionCount,
	})

	hostname, _ := os.Hostname()
	status.Hostname = hostname

	this.updateCPU(status)
	this.updateMem(status)
	this.updateLoad(status)
	this.updateDisk(status)
	status.UpdatedAt = time.Now().Unix()

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
	_, err = rpcClient.NodeRPC().UpdateNodeStatus(rpcClient.Context(), &pb.UpdateNodeStatusRequest{
		StatusJSON: jsonData,
	})
	if err != nil {
		remotelogs.Error("NODE_STATUS", "rpc UpdateNodeStatus() failed: "+err.Error())
		return
	}
}

// 更新CPU
func (this *NodeStatusExecutor) updateCPU(status *nodeconfigs.NodeStatus) {
	duration := time.Duration(0)
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
	})

	if this.cpuLogicalCount == 0 && this.cpuPhysicalCount == 0 {
		this.cpuUpdatedTime = time.Now()

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
	rootFS := ""
	rootTotal := uint64(0)
	if lists.ContainsString([]string{"darwin", "linux", "freebsd"}, runtime.GOOS) {
		for _, p := range partitions {
			if p.Mountpoint == "/" {
				rootFS = p.Fstype
				usage, _ := disk.Usage(p.Mountpoint)
				if usage != nil {
					rootTotal = usage.Total
				}
				break
			}
		}
	}

	total := rootTotal
	totalUsage := uint64(0)
	maxUsage := float64(0)
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
		}
		totalUsage += usage.Used
		if usage.UsedPercent >= maxUsage {
			maxUsage = usage.UsedPercent
			status.DiskMaxUsagePartition = partition.Mountpoint
		}
	}
	status.DiskTotal = total
	status.DiskUsage = float64(totalUsage) / float64(total)
	status.DiskMaxUsage = maxUsage / 100

	// 记录监控数据
	monitor.SharedValueQueue.Add(nodeconfigs.NodeValueItemDisk, maps.Map{
		"total":    status.DiskTotal,
		"usage":    status.DiskUsage,
		"maxUsage": status.DiskMaxUsage,
	})
}
