package mackerel

import (
	"context"
	"sync"
	"time"

	"go.opentelemetry.io/otel/api/core"
	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/api/unit"
	export "go.opentelemetry.io/otel/sdk/export/metric"
	"go.opentelemetry.io/otel/sdk/export/metric/aggregator"
	metricsdk "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/batcher/defaultkeys"
	"go.opentelemetry.io/otel/sdk/metric/controller/push"
	"go.opentelemetry.io/otel/sdk/metric/selector/simple"

	"github.com/mackerelio/mackerel-client-go"
)

const (
	UnitDimensionless = unit.Dimensionless
	UnitBytes         = unit.Bytes
	UnitMilliseconds  = unit.Milliseconds
)

// InstallNewPipeline instantiates a NewExportPipeline and registers it globally.
func InstallNewPipeline(opts ...Option) (*push.Controller, error) {
	pusher, err := NewExportPipeline(opts...)
	if err != nil {
		return nil, err
	}
	global.SetMeterProvider(pusher)
	return pusher, err
}

// NewExportPipeline sets up a complete export pipeline.
func NewExportPipeline(opts ...Option) (*push.Controller, error) {
	// There are few types in simple; inexpensive, sketch, exact.
	s := simple.NewWithExactMeasure()
	exporter, err := NewExporter(opts...)
	if err != nil {
		return nil, err
	}
	batcher := defaultkeys.New(s, metricsdk.NewDefaultLabelEncoder(), true)
	pusher := push.New(batcher, exporter, time.Minute)
	pusher.Start()
	return pusher, nil
}

// Option is function type that is passed to NewExporter function.
type Option func(*options)

type options struct {
	APIKey string
}

// WithAPIKey sets the Mackerel API Key.
func WithAPIKey(apiKey string) func(o *options) {
	return func(o *options) {
		o.APIKey = apiKey
	}
}

// Exporter is a stats exporter that uploads data to Mackerel.
type Exporter struct {
	quantile float64
	c        *mackerel.Client

	mu        sync.Mutex
	hostID    string
	graphDefs map[MetricName]struct{}
}

var _ export.Exporter = &Exporter{}

const defaultQuantile = 0.9

// NewExporter creates a new Exporter.
func NewExporter(opts ...Option) (*Exporter, error) {
	var o options
	for _, opt := range opts {
		opt(&o)
	}
	c := mackerel.NewClient(o.APIKey)
	return &Exporter{
		c: c,
	}, nil
}

func (e *Exporter) Export(ctx context.Context, a export.CheckpointSet) error {
	var metrics []*mackerel.HostMetricValue
	a.ForEach(func(r export.Record) {
		e.postHostAndGraphDefs(r)

		m := e.convertToHostMetric(r)
		if m == nil {
			return
		}
		metrics = append(metrics, m)
	})
	if err := e.c.PostHostMetricValues(metrics); err != nil {
		return err
	}
	return nil
}

func (e *Exporter) postHostAndGraphDefs(r export.Record) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	desc := r.Descriptor()
	name := NormalizeMetricName(desc.Name())
	if e.isGraphRegistered(name) {
		return nil
	}

	var res Resource
	labels := r.Labels().Ordered()
	if err := UnmarshalLabels(labels, &res); err != nil {
		return err
	}

	// TODO(lufia): if hostID is not set, it's the service metric.
	if e.hostID == "" {
		id, err := e.UpsertHost(&res)
		if err != nil {
			return err
		}
		e.hostID = id
	}

	metricClass := res.Mackerel.Metric.Class
	if metricClass == "" {
		metricClass = name
	}
	return e.registerGraph(MetricName(name))
}

func (e *Exporter) isGraphRegistered(name string) bool {
	for g := range e.graphDefs {
		if g.Match(name) {
			return true
		}
	}
	return false
}

func (e *Exporter) registerGraph(name MetricName) error {
	// TODO(lufia): implement
	e.graphDefs[name] = struct{}{}
	return nil
}

func (e *Exporter) convertToHostMetric(r export.Record) *mackerel.HostMetricValue {
	desc := r.Descriptor()
	name := NormalizeMetricName(desc.Name())
	aggr := r.Aggregator()
	kind := desc.NumberKind()

	m := metricValue(name, aggr, kind)
	return &mackerel.HostMetricValue{
		HostID:      e.hostID,
		MetricValue: m,
	}
}

func metricValue(name string, aggr export.Aggregator, kind core.NumberKind) *mackerel.MetricValue {
	var v interface{}

	// see https://github.com/open-telemetry/opentelemetry-go/blob/master/sdk/metric/selector/simple/simple.go
	if p, ok := aggr.(aggregator.Distribution); ok {
		// export.MeasureKind: MinMaxSumCount, Distribution, Points
		q, err := p.Quantile(defaultQuantile)
		if err != nil {
			return nil
		}
		v = q.AsInterface(kind)
	} else if p, ok := aggr.(aggregator.LastValue); ok {
		// export.GaugeKind: LastValue
		last, _, err := p.LastValue()
		if err != nil {
			return nil
		}
		v = last.AsInterface(kind)
	} else if p, ok := aggr.(aggregator.Sum); ok {
		// export.CounterKind: Sum
		sum, err := p.Sum()
		if err != nil {
			return nil
		}
		v = sum.AsInterface(kind)
	} else {
		return nil
	}

	return &mackerel.MetricValue{
		Name:  name,
		Time:  time.Now().Unix(),
		Value: v,
	}
}
