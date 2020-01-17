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
	c *mackerel.Client

	hosts     map[string]string // value is Mackerel's host ID
	graphDefs map[string]*mackerel.GraphDefsParam
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

type registration struct {
	res   *Resource
	def   *mackerel.GraphDefsParam
	value *mackerel.MetricValue
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
		id := customIdentifier(reg.res)
		if _, ok := e.hosts[id]; !ok {
			h, err := e.UpsertHost(reg.res)
			if err != nil {
				return err
			}
			e.hosts[id] = h
		}

		name := reg.def.Metrics[0].Name
		if g, ok := graphDefs[name]; ok {
			g.Metrics = append(g.Metrics, reg.def.Metrics[0])
		} else {
			graphDefs[name] = reg.def
		}

		hostID := e.hosts[id]
		metrics = append(metrics, &mackerel.HostMetricValue{
			HostID:      hostID,
			MetricValue: reg.value,
		})
		// TODO(lufia): post service metrics if host.id is not set and service.name is set.
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
	if e.graphDefs == nil {
		e.graphDefs = make(map[string]*mackerel.GraphDefsParam)
	}
	for k, v := range defs {
		if p, ok := e.graphDefs[k]; ok {
			p.Metrics = append(p.Metrics, v.Metrics...)
			continue
		}
		e.graphDefs[k] = v
	}
}

func (e *Exporter) convertToRegistration(r export.Record) (*registration, error) {
	desc := r.Descriptor()
	aggr := r.Aggregator()
	kind := desc.NumberKind()

	var res Resource
	labels := r.Labels().Ordered()
	if err := UnmarshalLabels(labels, &res); err != nil {
		return nil, err
	}

	name := NormalizeMetricName(desc.Name())
	gclass := NormalizeMetricName(res.Mackerel.Graph.Class)
	mclass := NormalizeMetricName(res.Mackerel.Metric.Class)
	switch {
	case mclass == "" && gclass == "":
		mclass = GeneralizeMetricName(name)
		gclass = mclass
	case mclass == "":
		s, err := AppendMetricName(gclass, mclass)
		if err != nil {
			return nil, err
		}
		mclass = s
	case gclass == "":
		gclass = mclass
	}
	if !MetricName(mclass).Match(name) {
		return nil, errMismatch
	}
	def := &mackerel.GraphDefsParam{
		Name: "custom." + gclass,
		Unit: GraphUnit(desc.Unit()),
		Metrics: []*mackerel.GraphDefsMetric{
			{Name: "custom." + mclass},
		},
	}

	m := metricValue(name, aggr, kind)
	return &registration{res: &res, def: def, value: m}, nil
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
