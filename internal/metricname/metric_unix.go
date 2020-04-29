// +build !windows

package metricname

// see https://mackerel.io/docs/entry/spec/metrics
var systemMetrics = []string{
	"loadavg1",
	"loadavg5",
	"loadavg15",
	"cpu.user.percentage",
	"cpu.iowait.percentage",
	"cpu.system.percentage",
	"cpu.idle.percentage",
	"cpu.nice.percentage",
	"cpu.irq.percentage",
	"cpu.softirq.percentage",
	"cpu.steal.percentage",
	"cpu.guest.percentage",
	"memory.used",
	"memory.available",
	"memory.total",
	"memory.swap_used",
	"memory.swap_cached",
	"memory.swap_total",
	"memory.free",
	"memory.buffers",
	"memory.cached",
	"memory.used",
	"memory.total",
	"memory.swap_used",
	"memory.swap_cached",
	"memory.swap_total",
	"disk.*.reads.delta",
	"disk.*.writes.delta",
	"interface.*.rxBytes.delta",
	"interface.*.txBytes.delta",
	"filesystem.*.size",
	"filesystem.*.used",
}
