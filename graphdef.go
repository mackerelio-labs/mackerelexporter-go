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

func metricNamePrefix(s string) string {
	a := strings.Split(s, metricNameSep)
	if len(a) == 0 {
		return ""
	}
	return strings.Join(a[:len(a)-1], metricNameSep)
}

// GraphDefOptions represents options for customizing Mackerel's Graph Definition.
type GraphDefOptions struct {
	Name      string
	Unit      unit.Unit
	Kind      core.NumberKind
	Quantiles []float64
}

var errMismatch = errors.New("mismatched metric names")

// NewGraphDef returns Mackerel's Graph Definition. Each names in arguments must be canonicalized.
func NewGraphDef(name string, kind export.MetricKind, opts GraphDefOptions) (*mackerel.GraphDefsParam, error) {
	if opts.Unit == "" {
		opts.Unit = UnitDimensionless
	}
	if kind == export.MeasureKind {
		name = JoinMetricName(name, "max") // Anything is fine
	}
	if opts.Name == "" {
		opts.Name = metricNamePrefix(name)
	}
	r := JoinMetricName(opts.Name, "*")
	if !MetricName(r).Match(name) {
		return nil, errMismatch
	}
	return &mackerel.GraphDefsParam{
		Name: opts.Name,
		Unit: graphUnit(opts.Unit, opts.Kind),
		Metrics: []*mackerel.GraphDefsMetric{
			{Name: r},
		},
	}, nil
}

// PercentileName returns "percentile_xx".
func PercentileName(q float64) string {
	return fmt.Sprintf("percentile_%.0f", math.Floor(q*100))
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

// CanonicalMetricName returns canonical metric name.
func CanonicalMetricName(s string) string {
	s = sanitizeMetricName(s)
	if isSystemMetric(s) {
		return s
	}
	return JoinMetricName("custom", s)
}

func sanitizeMetricName(s string) string {
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

// isSystemMetric returns whether s is system metric in Mackerel.
func isSystemMetric(s string) bool {
	for m := range systemMetricNames {
		if m.Match(s) {
			return true
		}
	}
	return false
}
