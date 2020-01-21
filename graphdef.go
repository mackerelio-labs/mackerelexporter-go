package mackerel

import (
	"errors"
	"fmt"

	"go.opentelemetry.io/otel/api/core"
	"go.opentelemetry.io/otel/api/unit"
	export "go.opentelemetry.io/otel/sdk/export/metric"

	"github.com/lufia/mackerelexporter-go/internal/metric"
	"github.com/mackerelio/mackerel-client-go"
)

const (
	unitDimensionless = unit.Dimensionless
	unitBytes         = unit.Bytes
	unitMilliseconds  = unit.Milliseconds
)

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
		opts.Unit = unitDimensionless
	}
	if kind == export.MeasureKind {
		name = metric.Join(name, "max") // Anything is fine
	}
	if opts.Name == "" {
		opts.Name = metric.Prefix(name)
	}
	r := metric.Join(opts.Name, "*")
	if !metric.Match(name, r) {
		return nil, errMismatch
	}
	return &mackerel.GraphDefsParam{
		Name:        opts.Name,
		DisplayName: opts.Name,
		Unit:        graphUnit(opts.Unit, opts.Kind),
		Metrics: []*mackerel.GraphDefsMetric{
			{Name: r, DisplayName: metricDisplayName(r)},
		},
	}, nil
}

func metricDisplayName(name string) string {
	a := metric.Split(name)
	if len(a) == 0 {
		return ""
	}
	var n int
	for _, s := range a {
		if s == "*" {
			n++
		}
	}
	if n == 0 {
		return a[len(a)-1]
	}
	return fmt.Sprintf("%%%d", n)
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
