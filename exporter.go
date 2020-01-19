package mackerel

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/api/core"
	"go.opentelemetry.io/otel/api/global"
	export "go.opentelemetry.io/otel/sdk/export/metric"
	"go.opentelemetry.io/otel/sdk/export/metric/aggregator"
	metricsdk "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/batcher/defaultkeys"
	"go.opentelemetry.io/otel/sdk/metric/controller/push"
	"go.opentelemetry.io/otel/sdk/metric/selector/simple"

	"github.com/mackerelio/mackerel-client-go"
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
	APIKey    string
	Quantiles []float64
}

// WithAPIKey sets the Mackerel API Key.
func WithAPIKey(apiKey string) func(o *options) {
	return func(o *options) {
		o.APIKey = apiKey
	}
}

// WithQuantiles sets quantiles for recording measure metrics.
// Each quantiles must be unique and its precision must be greater or equal than 0.01.
func WithQuantiles(quantiles []float64) func(o *options) {
	return func(o *options) {
		o.Quantiles = quantiles
	}
}

// Exporter is a stats exporter that uploads data to Mackerel.
type Exporter struct {
	c    *mackerel.Client
	opts *options

	hosts           map[string]string // value is Mackerel's host ID
	graphDefs       map[string]*mackerel.GraphDefsParam
	graphMetricDefs map[string]struct{}
}

var _ export.Exporter = &Exporter{}

var defaultQuantiles = []float64{0.99, 0.90, 0.75, 0.50, 0.25, 0.10}

// NewExporter creates a new Exporter.
func NewExporter(opts ...Option) (*Exporter, error) {
	var o options
	for _, opt := range opts {
		opt(&o)
	}
	if o.Quantiles == nil {
		o.Quantiles = defaultQuantiles
	}
	c := mackerel.NewClient(o.APIKey)
	return &Exporter{
		c:               c,
		opts:            &o,
		hosts:           make(map[string]string),
		graphDefs:       make(map[string]*mackerel.GraphDefsParam),
		graphMetricDefs: make(map[string]struct{}),
	}, nil
}

type registration struct {
	res      *Resource
	graphDef *mackerel.GraphDefsParam
	metrics  []*mackerel.MetricValue
}

func (e *Exporter) Export(ctx context.Context, a export.CheckpointSet) error {
	var regs []*registration
	a.ForEach(func(r export.Record) {
		reg, err := e.convertToRegistration(r)
		if err != nil {
			// TODO(lufia): output logs
			return
		}
		regs = append(regs, reg)
	})

	var metrics []*mackerel.HostMetricValue
	graphDefs := make(map[string]*mackerel.GraphDefsParam)
	for _, reg := range regs {
		// TODO(lufia): post service metrics if host.id is not set and service.name is set.
		id := reg.res.CustomIdentifier()
		if _, ok := e.hosts[id]; !ok {
			h, err := e.UpsertHost(reg.res)
			if err != nil {
				return err
			}
			e.hosts[id] = h
		}

		for _, m := range reg.graphDef.Metrics {
			if _, ok := e.graphMetricDefs[m.Name]; ok {
				// A graph is already registered; not need registration.
				continue
			}
			if g, ok := graphDefs[reg.graphDef.Name]; ok {
				g.Metrics = append(g.Metrics, m)
			} else {
				graphDefs[reg.graphDef.Name] = reg.graphDef
			}
		}

		hostID := e.hosts[id]
		for _, m := range reg.metrics {
			metrics = append(metrics, &mackerel.HostMetricValue{
				HostID:      hostID,
				MetricValue: m,
			})
		}
	}

	var defs []*mackerel.GraphDefsParam
	for _, d := range graphDefs {
		defs = append(defs, d)
	}
	if err := e.c.CreateGraphDefs(defs); err != nil {
		return err
	}
	e.mergeGraphDefs(graphDefs)

	if err := e.c.PostHostMetricValues(metrics); err != nil {
		return err
	}
	return nil
}

func (e *Exporter) mergeGraphDefs(defs map[string]*mackerel.GraphDefsParam) {
	for k, v := range defs {
		if p, ok := e.graphDefs[k]; ok {
			p.Metrics = append(p.Metrics, v.Metrics...)
		} else {
			e.graphDefs[k] = v
		}
		for _, m := range v.Metrics {
			e.graphMetricDefs[m.Name] = struct{}{}
		}
	}
}

func (e *Exporter) convertToRegistration(r export.Record) (*registration, error) {
	desc := r.Descriptor()
	kind := desc.NumberKind()

	var res Resource
	labels := r.Labels().Ordered()
	if err := UnmarshalLabels(labels, &res); err != nil {
		return nil, err
	}

	name := SanitizeMetricName(desc.Name())
	opts := GraphDefOptions{
		Name:       SanitizeMetricName(res.Mackerel.Graph.Class),
		MetricName: SanitizeMetricName(res.Mackerel.Metric.Class),
		Unit:       desc.Unit(),
		Kind:       kind,
		Quantiles:  e.opts.Quantiles,
	}
	g, err := NewGraphDef(name, desc.MetricKind(), opts)
	if err != nil {
		return nil, err
	}

	aggr := r.Aggregator()
	a := e.metricValues(name, aggr, kind)
	return &registration{res: &res, graphDef: g, metrics: a}, nil
}

func (e *Exporter) metricValues(name string, aggr export.Aggregator, kind core.NumberKind) []*mackerel.MetricValue {
	var a []*mackerel.MetricValue

	// see https://github.com/open-telemetry/opentelemetry-go/blob/master/sdk/metric/selector/simple/simple.go
	if p, ok := aggr.(aggregator.Distribution); ok {
		// export.MeasureKind: MinMaxSumCount, Distribution, Points
		if min, err := p.Min(); err == nil {
			a = append(a, metricValue(JoinMetricName(name, "min"), min.AsInterface(kind)))
		}
		if max, err := p.Max(); err == nil {
			a = append(a, metricValue(JoinMetricName(name, "max"), max.AsInterface(kind)))
		}
		for _, quantile := range e.opts.Quantiles {
			q, err := p.Quantile(quantile)
			if err == nil {
				return nil
			}
			qname := PercentileName(quantile)
			a = append(a, metricValue(JoinMetricName(name, qname), q.AsInterface(kind)))
		}
	} else if p, ok := aggr.(aggregator.LastValue); ok {
		// export.GaugeKind: LastValue
		if last, _, err := p.LastValue(); err == nil {
			a = append(a, metricValue(name, last.AsInterface(kind)))
		}
	} else if p, ok := aggr.(aggregator.Sum); ok {
		// export.CounterKind: Sum
		if sum, err := p.Sum(); err == nil {
			a = append(a, metricValue(name, sum.AsInterface(kind)))
		}
	}
	return a
}

func metricValue(name string, v ...interface{}) *mackerel.MetricValue {
	return &mackerel.MetricValue{
		Name:  JoinMetricName("custom", name),
		Time:  time.Now().Unix(),
		Value: v,
	}
}
