package mackerel

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/api/kv"
	"go.opentelemetry.io/otel/api/metric"
	export "go.opentelemetry.io/otel/sdk/export/metric"
	"go.opentelemetry.io/otel/sdk/export/metric/aggregator"
	"go.opentelemetry.io/otel/sdk/metric/controller/push"
	integrator "go.opentelemetry.io/otel/sdk/metric/integrator/simple"
	"go.opentelemetry.io/otel/sdk/metric/selector/simple"
	"go.opentelemetry.io/otel/sdk/resource"

	"github.com/lufia/mackerelexporter-go/internal/graphdef"
	"github.com/lufia/mackerelexporter-go/internal/metricname"
	"github.com/lufia/mackerelexporter-go/internal/tag"
	"github.com/mackerelio/mackerel-client-go"
)

// InstallNewPipeline instantiates a NewExportPipeline and registers it globally.
func InstallNewPipeline(opts ...Option) (*push.Controller, http.HandlerFunc, error) {
	pusher, handler, err := NewExportPipeline(opts...)
	if err != nil {
		return nil, nil, err
	}
	global.SetMeterProvider(pusher.Provider())
	return pusher, handler, err
}

// NewExportPipeline sets up a complete export pipeline.
func NewExportPipeline(opts ...Option) (*push.Controller, http.HandlerFunc, error) {
	// There are few types in simple; inexpensive, sketch, exact.
	s := simple.NewWithExactDistribution()
	exporter, err := NewExporter(opts...)
	if err != nil {
		return nil, nil, err
	}
	var o []push.Option
	o = append(o, push.WithPeriod(time.Minute))
	if len(exporter.opts.Tags) > 0 {
		res := resource.New(exporter.opts.Tags...)
		o = append(o, push.WithResource(res))
	}

	p := integrator.New(s, false)
	pusher := push.New(p, exporter, o...)
	pusher.Start()

	if h, _ := exporter.c.(http.Handler); h != nil {
		return pusher, h.ServeHTTP, nil
	}
	return pusher, nil, nil
}

// Option is function type that is passed to NewExporter function.
type Option func(*options)

type options struct {
	APIKey    string
	Quantiles []float64
	Hints     []string
	BaseURL   *url.URL
	Tags      []kv.KeyValue
	Debug     bool
}

// WithAPIKey sets the Mackerel API Key.
func WithAPIKey(apiKey string) Option {
	return func(o *options) {
		o.APIKey = apiKey
	}
}

// WithQuantiles sets quantiles for recording measure metrics.
// Each quantiles must be unique and its precision must be greater or equal than 0.01.
func WithQuantiles(quantiles []float64) Option {
	for _, q := range quantiles {
		if q < 0.0 || q > 1.0 {
			panic(aggregator.ErrInvalidQuantile)
		}
	}
	return func(o *options) {
		o.Quantiles = quantiles
	}
}

// WithHints sets hints for decision the name of the Graph Definition.
func WithHints(hints []string) Option {
	return func(o *options) {
		o.Hints = hints
	}
}

// WithBaseURL sets base URL for Mackerel API.
func WithBaseURL(baseURL *url.URL) Option {
	return func(o *options) {
		o.BaseURL = baseURL
	}
}

// WithResource sets resource tags.
func WithResource(tags ...kv.KeyValue) Option {
	return func(o *options) {
		o.Tags = tags
	}
}

// WithDebug enables logs for debugging.
func WithDebug() Option {
	return func(o *options) {
		o.Debug = true
	}
}

type mackerelClient interface {
	FindServices() ([]*mackerel.Service, error)
	CreateService(param *mackerel.CreateServiceParam) (*mackerel.Service, error)
	FindRoles(serviceName string) ([]*mackerel.Role, error)
	CreateRole(serviceName string, param *mackerel.CreateRoleParam) (*mackerel.Role, error)

	FindHosts(param *mackerel.FindHostsParam) ([]*mackerel.Host, error)
	CreateHost(param *mackerel.CreateHostParam) (string, error)
	UpdateHost(hostID string, param *mackerel.UpdateHostParam) (string, error)

	CreateGraphDefs(defs []*mackerel.GraphDefsParam) error
	PostHostMetricValues(metrics []*mackerel.HostMetricValue) error
	PostServiceMetricValues(name string, metrics []*mackerel.MetricValue) error
}

// Exporter is a stats exporter that uploads data to Mackerel.
type Exporter struct {
	c    mackerelClient
	opts *options

	hosts           map[string]string // value is Mackerel's host ID
	serviceRoles    map[string]map[string]struct{}
	graphDefs       map[string]*mackerel.GraphDefsParam
	graphMetricDefs map[string]struct{}
}

var _ export.Exporter = &Exporter{}

// NewExporter creates a new Exporter.
func NewExporter(opts ...Option) (*Exporter, error) {
	var o options
	for _, opt := range opts {
		opt(&o)
	}
	if o.Quantiles == nil {
		// This values equal to stdout exporter's values
		o.Quantiles = []float64{0.5, 0.9, 0.99}
	}
	var c mackerelClient = &handlerClient{}
	if o.APIKey != "" {
		p := mackerel.NewClient(o.APIKey)
		if o.BaseURL != nil {
			p.BaseURL = o.BaseURL
		}
		p.Verbose = o.Debug
		c = p
	}
	return &Exporter{
		c:               c,
		opts:            &o,
		hosts:           make(map[string]string),
		serviceRoles:    make(map[string]map[string]struct{}),
		graphDefs:       make(map[string]*mackerel.GraphDefsParam),
		graphMetricDefs: make(map[string]struct{}),
	}, nil
}

type (
	registration struct {
		res      *tag.Resource
		graphDef *mackerel.GraphDefsParam
		metrics  []*mackerel.MetricValue
	}

	customIdentifier string
	serviceName      string
)

// Export exports the provide metric record to Mackerel.
func (e *Exporter) Export(ctx context.Context, a export.CheckpointSet) error {
	var regs []*registration
	a.ForEach(func(r export.Record) error {
		reg, err := e.convertToRegistration(r, r.Resource())
		if err != nil {
			return err
		}
		regs = append(regs, reg)
		return nil
	})

	var (
		hostMetrics    []*mackerel.HostMetricValue
		serviceMetrics = make(map[string][]*mackerel.MetricValue)
		graphDefs      = make(map[string]*mackerel.GraphDefsParam)
	)
	for _, reg := range regs {
		switch t := metricType(reg.res); s := t.(type) {
		case customIdentifier:
			id := string(s)
			if _, ok := e.hosts[id]; !ok {
				h, err := e.upsertHost(reg.res)
				if err != nil {
					return err
				}
				e.hosts[id] = h
			}

			hostID := e.hosts[id]
			for _, m := range reg.metrics {
				hostMetrics = append(hostMetrics, &mackerel.HostMetricValue{
					HostID:      hostID,
					MetricValue: m,
				})
			}
		case serviceName:
			name := string(s)
			if err := e.registerService(name); err != nil {
				return err
			}
			serviceMetrics[name] = append(serviceMetrics[name], reg.metrics...)
		default:
			continue
		}

		if reg.graphDef != nil {
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
		}
	}

	var defs []*mackerel.GraphDefsParam
	for _, d := range graphDefs {
		defs = append(defs, d)
	}
	if len(defs) > 0 {
		if err := e.c.CreateGraphDefs(defs); err != nil {
			return fmt.Errorf("can't create graph-defs: %w", err)
		}
		e.mergeGraphDefs(graphDefs)
	}

	if len(hostMetrics) > 0 {
		if err := e.c.PostHostMetricValues(hostMetrics); err != nil {
			return fmt.Errorf("can't post host metrics: %w", err)
		}
	}
	for s, a := range serviceMetrics {
		if err := e.c.PostServiceMetricValues(s, a); err != nil {
			return fmt.Errorf("can't post service metrics: %w", err)
		}
	}
	return nil
}

func metricType(res *tag.Resource) interface{} {
	if s := res.CustomIdentifier(); s != "" {
		return customIdentifier(s)
	}
	if s := res.ServiceName(); s != "" {
		return serviceName(s)
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

func (e *Exporter) convertToRegistration(r export.Record, res *resource.Resource) (*registration, error) {
	var reg registration
	desc := r.Descriptor()
	kind := desc.NumberKind()

	var t tag.Resource
	labels := append(r.Labels().ToSlice(), res.Attributes()...)
	if err := tag.UnmarshalTags(labels, &t); err != nil {
		return nil, err
	}
	reg.res = &t

	// TODO(lufia): Enforce the metric to be the custom metric if hint is exist
	name := metricname.Canonical(desc.Name())
	hint := e.lookupHint(desc.Name())
	aggr := r.Aggregator()
	reg.metrics = e.metricValues(name, aggr, kind)

	if !strings.HasPrefix(name, "custom.") {
		return &reg, nil
	}
	opts := graphdef.Options{
		Name:      hint,
		Unit:      desc.Unit(),
		Kind:      kind,
		Quantiles: e.opts.Quantiles,
	}
	g, err := graphdef.New(name, desc.MetricKind(), opts)
	if err != nil {
		return nil, err
	}
	reg.graphDef = g

	return &reg, nil
}

func (e *Exporter) lookupHint(name string) string {
	for _, s := range e.opts.Hints {
		if metricname.Match(name, s) {
			return metricname.Canonical(s)
		}
	}
	return ""
}

func (e *Exporter) metricValues(name string, aggr export.Aggregator, kind metric.NumberKind) []*mackerel.MetricValue {
	var a []*mackerel.MetricValue

	// see https://github.com/open-telemetry/opentelemetry-go/blob/master/sdk/metric/selector/simple/simple.go
	if p, ok := aggr.(aggregator.Distribution); ok {
		// metric.Value{Record|Obserb}erKind: MinMaxSumCount, Distribution, Points
		if min, err := p.Min(); err == nil {
			a = append(a, metricValue(metricname.Join(name, "min"), min.AsInterface(kind)))
		}
		if max, err := p.Max(); err == nil {
			a = append(a, metricValue(metricname.Join(name, "max"), max.AsInterface(kind)))
		}
		for _, quantile := range e.opts.Quantiles {
			q, err := p.Quantile(quantile)
			if err != nil {
				continue
			}
			qname := metricname.Percentile(quantile)
			a = append(a, metricValue(metricname.Join(name, qname), q.AsInterface(kind)))
		}
	} else if p, ok := aggr.(aggregator.LastValue); ok {
		// Where this aggregator is used in?
		if last, _, err := p.LastValue(); err == nil {
			a = append(a, metricValue(name, last.AsInterface(kind)))
		}
	} else if p, ok := aggr.(aggregator.Sum); ok {
		// metric.CounterKind, etc: Sum
		if sum, err := p.Sum(); err == nil {
			a = append(a, metricValue(name, sum.AsInterface(kind)))
		}
	}
	return a
}

func metricValue(name string, v interface{}) *mackerel.MetricValue {
	return &mackerel.MetricValue{
		Name:  name,
		Time:  time.Now().Unix(),
		Value: v,
	}
}

func (e *Exporter) Handler() http.Handler {
	h, _ := e.c.(http.Handler)
	return h
}
