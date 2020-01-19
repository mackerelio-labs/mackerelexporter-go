package mackerel

import (
	"errors"
	"strings"

	"go.opentelemetry.io/otel/api/core"
	"go.opentelemetry.io/otel/api/unit"

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

// GraphDefOptions represents options for customizing Mackerel's Graph Definition.
type GraphDefOptions struct {
	Name       string
	MetricName string
	Unit       unit.Unit
	Kind       core.NumberKind
}

// NewGraphDef returns Mackerel's Graph Definition that has only one metric in Metrics field.
// Each names in arguments must be normalized.
func NewGraphDef(name string, opts GraphDefOptions) (*mackerel.GraphDefsParam, error) {
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
	return &mackerel.GraphDefsParam{
		Name: "custom." + opts.Name,
		Unit: graphUnit(opts.Unit, opts.Kind),
		Metrics: []*mackerel.GraphDefsMetric{
			{Name: "custom." + opts.MetricName},
		},
	}, nil
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

// NormalizeMetricName returns normalized s.
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
