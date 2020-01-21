// +build windows

package metric

// see https://mackerel.io/docs/entry/spec/metrics
var systemMetrics = []string{
	"processor_queue_length",
	"cpu.user.percentage",
	"cpu.system.percentage",
	"cpu.idle.percentage",
	"memory.free",
	"memory.used",
	"memory.total",
	"memory.pagefile_free",
	"memory.pagefile_total",
	"disk.*.reads.delta",
	"disk.*.writes.delta",
	"interface.*.rxBytes.delta",
	"interface.*.txBytes.delta",
	"filesystem.*.size",
	"filesystem.*.used",
}
