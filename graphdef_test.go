package mackerel

import (
	"reflect"
	"testing"

	"github.com/mackerelio/mackerel-client-go"
	"go.opentelemetry.io/otel/api/core"
	export "go.opentelemetry.io/otel/sdk/export/metric"
)

func TestNewGraphDef(t *testing.T) {
	tests := []struct {
		desc string
		kind export.MetricKind
		name string
		opts GraphDefOptions
		want *mackerel.GraphDefsParam
	}{
		{
			desc: "simple_counter",
			kind: export.CounterKind,
			name: "custom.ether0.txBytes",
			opts: GraphDefOptions{},
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
			kind: export.CounterKind,
			name: "custom.ether0.txBytes",
			opts: GraphDefOptions{
				Name: "custom.#",
				Kind: core.Float64NumberKind,
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
			kind: export.MeasureKind,
			name: "custom.http.latency",
			opts: GraphDefOptions{},
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
			kind: export.MeasureKind,
			name: "custom.http.index.latency",
			opts: GraphDefOptions{
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
			g, err := NewGraphDef(tt.name, tt.kind, tt.opts)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(g, tt.want) {
				t.Errorf("NewGraphDef(%s, %v, opts) = %v; want %v", tt.name, tt.kind, g, tt.want)
			}
		})
	}
}
