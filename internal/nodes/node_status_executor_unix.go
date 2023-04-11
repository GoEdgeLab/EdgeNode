//go:build !windows

package nodes

import (
	"github.com/TeaOSLab/EdgeCommon/pkg/nodeconfigs"
	"github.com/TeaOSLab/EdgeNode/internal/monitor"
	"github.com/iwind/TeaGo/maps"
	"github.com/shirou/gopsutil/v3/load"
	"github.com/shirou/gopsutil/v3/mem"
	"runtime"
	"runtime/debug"
)

// 更新内存
func (this *NodeStatusExecutor) updateMem(status *nodeconfigs.NodeStatus) {
	stat, err := mem.VirtualMemory()
	if err != nil {
		return
	}

	// 重新计算内存
	if stat.Total > 0 {
		stat.Used = stat.Total - stat.Free - stat.Buffers - stat.Cached
		status.MemoryUsage = float64(stat.Used) / float64(stat.Total)
	}

	status.MemoryTotal = stat.Total

	// 记录监控数据
	monitor.SharedValueQueue.Add(nodeconfigs.NodeValueItemMemory, maps.Map{
		"usage": status.MemoryUsage,
		"total": status.MemoryTotal,
		"used":  stat.Used,
	})

	// 内存严重不足时自动释放内存
	if stat.Total > 0 {
		var minFreeMemory = stat.Total / 8
		if minFreeMemory > 1<<30 {
			minFreeMemory = 1 << 30
		}
		if stat.Available > 0 && stat.Available < minFreeMemory {
			runtime.GC()
			debug.FreeOSMemory()
		}
	}
}

// 更新负载
func (this *NodeStatusExecutor) updateLoad(status *nodeconfigs.NodeStatus) {
	stat, err := load.Avg()
	if err != nil {
		status.Error = err.Error()
		return
	}
	if stat == nil {
		status.Error = "load is nil"
		return
	}
	status.Load1m = stat.Load1
	status.Load5m = stat.Load5
	status.Load15m = stat.Load15

	// 记录监控数据
	monitor.SharedValueQueue.Add(nodeconfigs.NodeValueItemLoad, maps.Map{
		"load1m":  status.Load1m,
		"load5m":  status.Load5m,
		"load15m": status.Load15m,
	})
}
