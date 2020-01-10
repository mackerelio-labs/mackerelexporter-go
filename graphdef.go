package mackerel

import (
	"strings"
)

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
