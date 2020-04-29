package metricname

import (
	"fmt"
	"math"
	"strings"
)

// OpenTelemetry naming
// https://github.com/open-telemetry/opentelemetry-specification/blob/master/specification/api-metrics-user.md

const metricNameSep = "."

// Join joins any number of name elements int a single name.
func Join(elem ...string) string {
	a := make([]string, len(elem))
	for i, s := range elem {
		a[i] = s
	}
	return strings.Join(a, metricNameSep)
}

// Split slices s into all substrings
func Split(s string) []string {
	return strings.Split(s, metricNameSep)
}

// Sanitize sanitizes s.
func Sanitize(s string) string {
	sanitize := func(c rune) rune {
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
	return strings.Map(sanitize, s)
}

// Match evaluates that s is matched to pattern.
func Match(s, pattern string) bool {
	a := Split(s)
	expr := Split(pattern)
	if len(a) != len(expr) {
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

// Prefix splits s immediately following the final dot.
func Prefix(s string) string {
	a := strings.Split(s, metricNameSep)
	if len(a) == 0 {
		return ""
	}
	return strings.Join(a[:len(a)-1], metricNameSep)
}

// Percentile returns "percentile_xx".
func Percentile(q float64) string {
	return fmt.Sprintf("percentile_%.0f", math.Floor(q*100))
}

// Canonical returns canonical metric name.
func Canonical(s string) string {
	s = Sanitize(s)
	if isSystemMetric(s) {
		return s
	}
	return Join("custom", s)
}

var systemMetricNames map[string]struct{}

func init() {
	systemMetricNames = make(map[string]struct{})
	for _, s := range systemMetrics {
		systemMetricNames[s] = struct{}{}
	}
}

// isSystemMetric returns whether s is system metric in Mackerel.
func isSystemMetric(s string) bool {
	for m := range systemMetricNames {
		if Match(s, m) {
			return true
		}
	}
	return false
}
