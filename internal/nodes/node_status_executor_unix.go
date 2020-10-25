// +build !windows

package nodes

import (
	"github.com/TeaOSLab/EdgeCommon/pkg/nodeconfigs"
	"github.com/shirou/gopsutil/load"
	"github.com/shirou/gopsutil/mem"
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
}
