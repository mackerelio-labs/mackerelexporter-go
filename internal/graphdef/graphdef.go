package graphdef

import (
	"errors"
	"fmt"

	"go.opentelemetry.io/otel/api/core"
	"go.opentelemetry.io/otel/api/metric"
	"go.opentelemetry.io/otel/api/unit"

	metricname "github.com/lufia/mackerelexporter-go/internal/metric"
	"github.com/mackerelio/mackerel-client-go"
)

const (
	unitDimensionless = unit.Dimensionless
	unitBytes         = unit.Bytes
	unitMilliseconds  = unit.Milliseconds
)

// Options represents options for customizing Mackerel's Graph Definition.
type Options struct {
	Name      string
	Unit      unit.Unit
	Kind      core.NumberKind
	Quantiles []float64
}

var errMismatch = errors.New("mismatched metric names")

// New returns Mackerel's Graph Definition. Each names in arguments must be canonicalized.
func New(name string, kind metric.Kind, opts Options) (*mackerel.GraphDefsParam, error) {
	if opts.Unit == "" {
		opts.Unit = unitDimensionless
	}
	if kind == metric.MeasureKind {
		name = metricname.Join(name, "max") // Anything is fine
	}
	if opts.Name == "" {
		opts.Name = metricname.Prefix(name)
	}
	r := metricname.Join(opts.Name, "*")
	if !metricname.Match(name, r) {
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
	a := metricname.Split(name)
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
