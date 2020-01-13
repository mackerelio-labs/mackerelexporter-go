package mackerel

import (
	"context"
	"strings"
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

var (
	// see https://github.com/open-telemetry/opentelemetry-specification/blob/master/specification/data-resource-semantic-conventions.md
	keyHostID   = core.Key("host.id")
	keyHostName = core.Key("host.name")
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

func (e *Exporter) convertToHostMetric(r export.Record) *mackerel.HostMetricValue {
	desc := r.Descriptor()
	name := cleanName(desc.Name())
	aggr := r.Aggregator()
	kind := desc.NumberKind()

	// TODO(lufia): desc.Description will be used for graph-def.

	labels := r.Labels().Ordered()
	meta := hostMetaFromLabels(labels)
	hostID := meta[keyHostID]
	m := metricValue(name, aggr, kind)
	return &mackerel.HostMetricValue{
		HostID:      hostID,
		MetricValue: m,
	}
}

func hostMetaFromLabels(labels []core.KeyValue) map[core.Key]string {
	m := make(map[core.Key]string)
	for _, kv := range labels {
		if !kv.Key.Defined() {
			continue
		}
		m[kv.Key] = kv.Value.Emit()
	}
	return m
}

// Deprecated: We might use labels; {class=custom.xxx.#.*.name}
func metricName(d *export.Descriptor) string {
	s1, s2 := SplitGraphName(d.Name())
	return strings.Join([]string{s1, s2}, ".")
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

type Host struct {
	Name             string
	CustomIdentifier string
	Meta             HostMeta

	//Roles            Roles
	//Interfaces       []Interface
}

type HostMeta struct {
	AgentVersion string
	AgentName    string
	CPUName      string
	CPUMHz       int

	//BlockDevice   BlockDevice
	//Filesystem    FileSystem
	//Memory        Memory
	//Cloud         *Cloud
}

func (e *Exporter) RegisterHost(h *Host) (string, error) {
	id, err := e.lookupHost(h)
	if err != nil {
		return "", err
	}
	if id != "" {
		// TODO(lufia): we should update a host
		return id, nil // The host was already registered
	}

	cpu0 := map[string]interface{}{
		"model_name": h.Meta.CPUName,
		"mhz":        h.Meta.CPUMHz,
	}
	param := mackerel.CreateHostParam{
		Name:             h.Name,
		CustomIdentifier: h.CustomIdentifier,
		Meta: mackerel.HostMeta{
			AgentVersion: h.Meta.AgentVersion,
			AgentName:    h.Meta.AgentName,
			CPU:          mackerel.CPU{cpu0},
			Kernel: map[string]string{
				"os":      "Plan 9",
				"release": "4e",
				"version": "2000",
			},
		},
	}
	return e.c.CreateHost(&param)
}

func (e *Exporter) lookupHost(h *Host) (string, error) {
	if h.CustomIdentifier != "" {
		a, err := e.c.FindHosts(&mackerel.FindHostsParam{
			CustomIdentifier: h.CustomIdentifier,
		})
		if err != nil {
			return "", err
		}
		if len(a) > 0 {
			return a[0].ID, nil
		}
	}
	return "", nil // not found
}
