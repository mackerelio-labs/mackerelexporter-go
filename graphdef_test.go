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
				Name: "custom.ether0",
				Unit: "integer",
				Metrics: []*mackerel.GraphDefsMetric{
					{Name: "custom.ether0.*"},
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
				Name: "custom.#",
				Unit: "float",
				Metrics: []*mackerel.GraphDefsMetric{
					{Name: "custom.#.*"},
				},
			},
		},
		{
			desc: "simple_measure",
			kind: export.MeasureKind,
			name: "custom.http.latency",
			opts: GraphDefOptions{},
			want: &mackerel.GraphDefsParam{
				Name: "custom.http.latency",
				Unit: "integer",
				Metrics: []*mackerel.GraphDefsMetric{
					{Name: "custom.http.latency.*"},
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

func TestSanitizeMetricName(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{name: "abc.def", want: "abc.def"},
		{name: "custom.#.*.t-x_a", want: "custom.#.*.t-x_a"},
		{name: "aaa.$!.bb", want: "aaa.__.bb"},
	}
	for _, tt := range tests {
		s := sanitizeMetricName(tt.name)
		if s != tt.want {
			t.Errorf("sanitizeMetricName(%q) = %q; want %q", tt.name, s, tt.want)
		}
	}
}

func TestIsSystemMetric(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{name: "memory.used", want: true},
		{name: "memory.total", want: true},
		{name: "memory.xxx", want: false},
		{name: "xmemory.used", want: false},
		{name: "memory.usedx", want: false},
		{name: "filesystem.sdC0.size", want: true},
	}
	for _, tt := range tests {
		v := isSystemMetric(tt.name)
		if v != tt.want {
			t.Errorf("isSystemMetric(%q) = %t; want %t", tt.name, v, tt.want)
		}
	}
}

func TestGraphName(t *testing.T) {
	tests := []struct {
		name      MetricName
		matches   []string
		unmatches []string
	}{
		{
			name: "memory.avail",
			matches: []string{
				"memory.avail",
			},
			unmatches: []string{
				"memory.usage",
				"memory",
				"memory.avail.min",
			},
		},
		{
			name: "custom.cpu.#.user",
			matches: []string{
				"custom.cpu.x1.user",
				"custom.cpu.x2.user",
			},
			unmatches: []string{
				"custom.memory.x3.user",
				"custom.cpu.x3.sys",
				"custom.cpu.x3.user.min",
			},
		},
		{
			name: "custom.cpu.*.user",
			matches: []string{
				"custom.cpu.x1.user",
				"custom.cpu.x2.user",
			},
			unmatches: []string{
				"custom.memory.x3.user",
				"custom.cpu.x3.sys",
				"custom.cpu.x3.user.min",
			},
		},
	}
	for _, tt := range tests {
		t.Run("matches", func(t *testing.T) {
			for _, s := range tt.matches {
				if ok := tt.name.Match(s); !ok {
					t.Errorf("%q.Match(%q) = %t", tt.name, s, ok)
				}
			}
		})
		t.Run("unmatches", func(t *testing.T) {
			for _, s := range tt.unmatches {
				if ok := tt.name.Match(s); ok {
					t.Errorf("%q.Match(%q) = %t", tt.name, s, ok)
				}
			}
		})
	}
}
