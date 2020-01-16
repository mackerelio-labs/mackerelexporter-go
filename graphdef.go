package mackerel

import (
	"strings"
)

// OpenTelemetry naming
// https://github.com/open-telemetry/opentelemetry-specification/blob/master/specification/api-metrics-user.md

// SplitGraphName splits "a/b" into groupName, metricName.
func SplitGraphName(s string) (groupName, metricName string) {
	i := strings.LastIndex(s, "/")
	if i < 0 {
		return "", cleanName(s)
	}
	return cleanName(s[:i]), cleanName(s[i+1:])
}

func cleanName(s string) string {
	s = strings.ReplaceAll(s, ".", "_")
	return strings.ReplaceAll(s, "/", ".")
}

var systemNames = []string{
	"memory.used",
	"memory.available",
	"memory.total",
	"memory.swap_used",
	"memory.swap_cached",
	"memory.swap_total",
}

// IsSystemMetric returns whether s is system metric in Mackerel.
func IsSystemMetric(s string) bool {
	s = cleanName(s)
	for _, m := range systemNames {
		if s == m {
			return true
		}
	}
	return false
}

const (
	graphNameSep = "."
)

type GraphName string

func (g GraphName) Match(s string) bool {
	expr := strings.Split(string(g), graphNameSep)
	a := strings.Split(s, graphNameSep)
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
