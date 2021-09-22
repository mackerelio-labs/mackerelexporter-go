package graphdef

import (
	"errors"
	"fmt"

	"go.opentelemetry.io/otel/metric/number"
	"go.opentelemetry.io/otel/metric/sdkapi"
	"go.opentelemetry.io/otel/metric/unit"

	"github.com/mackerelio-labs/mackerelexporter-go/internal/metricname"
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
	Kind      number.Kind
	Quantiles []float64
}

var errMismatch = errors.New("mismatched metric names")

// New returns Mackerel's Graph Definition. Each names in arguments must be canonicalized.
func New(name string, kind sdkapi.InstrumentKind, opts Options) (*mackerel.GraphDefsParam, error) {
	if opts.Unit == "" {
		opts.Unit = unitDimensionless
	}
	if kind == sdkapi.HistogramInstrumentKind {
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

func graphUnit(u unit.Unit, kind number.Kind) string {
	switch u {
	case unit.Bytes:
		return "bytes"
	case unit.Dimensionless, unit.Milliseconds:
		if kind == number.Float64Kind {
			return "float"
		}
		return "integer"
	default:
		if kind == number.Float64Kind {
			return "float"
		}
		return "float"
	}
}
