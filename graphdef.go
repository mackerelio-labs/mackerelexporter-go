package mackerel

import (
	"errors"
	"fmt"
	"math"
	"strings"

	"go.opentelemetry.io/otel/api/core"
	"go.opentelemetry.io/otel/api/unit"
	export "go.opentelemetry.io/otel/sdk/export/metric"

	"github.com/mackerelio/mackerel-client-go"
)

// OpenTelemetry naming
// https://github.com/open-telemetry/opentelemetry-specification/blob/master/specification/api-metrics-user.md

const (
	UnitDimensionless = unit.Dimensionless
	UnitBytes         = unit.Bytes
	UnitMilliseconds  = unit.Milliseconds

	metricNameSep = "."
)

// JoinMetricName joins any number of name elements int a single name.
func JoinMetricName(elem ...string) string {
	a := make([]string, len(elem))
	for i, s := range elem {
		a[i] = s
	}
	return strings.Join(a, metricNameSep)
}

// GraphDefOptions represents options for customizing Mackerel's Graph Definition.
type GraphDefOptions struct {
	Name       string
	MetricName string
	Unit       unit.Unit
	Kind       core.NumberKind
	Quantiles  []float64
}

// NewGraphDef returns Mackerel's Graph Definition. Each names in arguments must be sanitized.
func NewGraphDef(name string, kind export.MetricKind, opts GraphDefOptions) (*mackerel.GraphDefsParam, error) {
	if opts.Unit == "" {
		opts.Unit = UnitDimensionless
	}
	switch {
	case opts.MetricName == "" && opts.Name == "":
		opts.MetricName = generalizeMetricName(name)
		opts.Name = opts.MetricName
	case opts.MetricName == "" && opts.Name != "":
		s, err := replaceMetricNamePrefix(name, opts.Name)
		if err != nil {
			return nil, err
		}
		opts.MetricName = s
	case opts.MetricName != "" && opts.Name == "":
		opts.Name = opts.MetricName
	}
	if !MetricName(opts.MetricName).Match(name) {
		return nil, errMismatch
	}
	g := &mackerel.GraphDefsParam{
		Name: JoinMetricName("custom", opts.Name),
		Unit: graphUnit(opts.Unit, opts.Kind),
	}
	if kind == export.MeasureKind {
		g.Metrics = measureMetrics(opts.MetricName, opts.Quantiles)
	} else {
		g.Metrics = []*mackerel.GraphDefsMetric{
			{Name: JoinMetricName("custom", opts.MetricName)},
		}
	}
	return g, nil
}

// PercentileName returns "percentile_xx".
func PercentileName(q float64) string {
	return fmt.Sprintf("percentile_%.0f", math.Floor(q*100))
}

func measureMetrics(name string, quantiles []float64) []*mackerel.GraphDefsMetric {
	suffixes := []string{"min", "max"}
	for _, q := range quantiles {
		suffixes = append(suffixes, PercentileName(q))
	}
	var a []*mackerel.GraphDefsMetric
	for _, s := range suffixes {
		a = append(a, &mackerel.GraphDefsMetric{
			Name: JoinMetricName("custom", name, s),
		})
	}
	return a
}

func graphUnit(u unit.Unit, kind core.NumberKind) string {
	switch u {
	case unit.Bytes:
		return "bytes"
	case unit.Dimensionless, unit.Milliseconds:
		if kind == core.Float64NumberKind {
			return "float"
		}
		return "integer"
	default:
		if kind == core.Float64NumberKind {
			return "float"
		}
		return "float"
	}
}

var errMismatch = errors.New("mismatched metric names")

// replaceMetricNamePrefix returns prefix + rest of s.
func replaceMetricNamePrefix(s, prefix string) (string, error) {
	a1 := strings.Split(prefix, metricNameSep)
	a2 := strings.Split(s, metricNameSep)
	if len(a1) > len(a2) {
		return "", errMismatch
	}
	t := strings.Join(a2[:len(a1)], metricNameSep)
	if !MetricName(prefix).Match(t) {
		return "", errMismatch
	}
	copy(a2[:len(a1)], a1)
	return strings.Join(a2, metricNameSep), nil
}

// generalizeMetricName generalize "a.b" to "a.*" if s don't contain wildcards.
func generalizeMetricName(s string) string {
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

// SanitizeMetricName returns sanitized s.
func SanitizeMetricName(s string) string {
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

var systemMetricNames map[MetricName]struct{}

func init() {
	systemMetricNames = make(map[MetricName]struct{})
	for _, s := range systemMetrics {
		systemMetricNames[s] = struct{}{}
	}
}

// IsSystemMetric returns whether s is system metric in Mackerel.
func IsSystemMetric(s string) bool {
	for m := range systemMetricNames {
		if m.Match(s) {
			return true
		}
	}
	return false
}
