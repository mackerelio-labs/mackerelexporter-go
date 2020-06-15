package graphdef

import (
	"reflect"
	"testing"

	"github.com/mackerelio/mackerel-client-go"
	"go.opentelemetry.io/otel/api/metric"
)

func TestNew(t *testing.T) {
	tests := []struct {
		desc string
		kind metric.Kind
		name string
		opts Options
		want *mackerel.GraphDefsParam
	}{
		{
			desc: "simple_counter",
			kind: metric.CounterKind,
			name: "custom.ether0.txBytes",
			opts: Options{},
			want: &mackerel.GraphDefsParam{
				Name:        "custom.ether0",
				DisplayName: "custom.ether0",
				Unit:        "integer",
				Metrics: []*mackerel.GraphDefsMetric{
					{
						Name:        "custom.ether0.*",
						DisplayName: "%1",
					},
				},
			},
		},
		{
			desc: "counter_with_options",
			kind: metric.CounterKind,
			name: "custom.ether0.txBytes",
			opts: Options{
				Name: "custom.#",
				Kind: metric.Float64NumberKind,
			},
			want: &mackerel.GraphDefsParam{
				Name:        "custom.#",
				DisplayName: "custom.#",
				Unit:        "float",
				Metrics: []*mackerel.GraphDefsMetric{
					{
						Name:        "custom.#.*",
						DisplayName: "%1",
					},
				},
			},
		},
		{
			desc: "simple_measure",
			kind: metric.ValueRecorderKind,
			name: "custom.http.latency",
			opts: Options{},
			want: &mackerel.GraphDefsParam{
				Name:        "custom.http.latency",
				DisplayName: "custom.http.latency",
				Unit:        "integer",
				Metrics: []*mackerel.GraphDefsMetric{
					{
						Name:        "custom.http.latency.*",
						DisplayName: "%1",
					},
				},
			},
		},
		{
			desc: "multiple_wildcard",
			kind: metric.ValueRecorderKind,
			name: "custom.http.index.latency",
			opts: Options{
				Name: "custom.http.#.*",
			},
			want: &mackerel.GraphDefsParam{
				Name:        "custom.http.#.*",
				DisplayName: "custom.http.#.*",
				Unit:        "integer",
				Metrics: []*mackerel.GraphDefsMetric{
					{
						Name:        "custom.http.#.*.*",
						DisplayName: "%2",
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			g, err := New(tt.name, tt.kind, tt.opts)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(g, tt.want) {
				t.Errorf("New(%s, %v, opts) = %v; want %v", tt.name, tt.kind, g, tt.want)
			}
		})
	}
}
