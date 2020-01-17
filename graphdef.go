package mackerel

import (
	"strings"
)

// OpenTelemetry naming
// https://github.com/open-telemetry/opentelemetry-specification/blob/master/specification/api-metrics-user.md

const (
	metricNameSep = "."
)

// GeneralizeMetricName generalize "a.b" to "a.*" if s don't contain wildcards.
func GeneralizeMetricName(s string) string {
	if s == "" {
		return ""
	}
	a := strings.Split(s, metricNameSep)
	for _, stem := range a {
		if stem == "*" {
			return s
		}
	}
	if a[len(a)-1] == "#" {
		return s
	}
	a[len(a)-1] = "*"
	return strings.Join(a, metricNameSep)
}

func NormalizeMetricName(s string) string {
	normalize := func(c rune) rune {
		switch {
		case c >= '0' && c <= '9':
			return c
		case c >= 'a' && c <= 'z':
			return c
		case c >= 'A' && c <= 'Z':
			return c
		case c == '-' || c == '_' || c == '.' || c == '#' || c == '*':
			return c
		default:
			return '_'
		}
	}
	return strings.Map(normalize, s)
}

type MetricName string

func (g MetricName) Match(s string) bool {
	expr := strings.Split(string(g), metricNameSep)
	a := strings.Split(s, metricNameSep)
	if len(expr) != len(a) {
		return false
	}
	for i := range expr {
		if expr[i] == "#" || expr[i] == "*" {
			continue
		}
		if expr[i] != a[i] {
			return false
		}
	}
	return true
}

// see https://mackerel.io/docs/entry/spec/metrics
var systemMetrics = []MetricName{
	/* Linux */
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

	/* Windows */
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

var systemMetricNames map[MetricName]struct{}

func init() {
	systemMetricNames = make(map[MetricName]struct{})
	for _, s := range systemMetrics {
		systemMetricNames[s] = struct{}{}
	}
}

// IsSystemMetric returns whether s is system metric in Mackerel.
func IsSystemMetric(s string) bool {
	s = NormalizeMetricName(s)
	for m := range systemMetricNames {
		if m.Match(s) {
			return true
		}
	}
	return false
}
