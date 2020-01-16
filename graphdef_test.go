package mackerel

import (
	"testing"
)

func TestSplitGraphName(t *testing.T) {
	tests := []struct {
		name string

		group  string
		metric string
	}{
		{name: "memory/available", group: "memory", metric: "available"},
		{name: "memory/", group: "memory", metric: ""},
		{name: "/available", group: "", metric: "available"},
		{name: "os.name", group: "", metric: "os_name"},
		{name: "mackerel.io/name", group: "mackerel_io", metric: "name"},
	}
	for _, tt := range tests {
		s1, s2 := SplitGraphName(tt.name)
		if s1 != tt.group {
			t.Errorf("SplitGraphName(%q) = (%q, %q); want (%q, %q)", tt.name, s1, s2, tt.group, tt.metric)
		}
		if s2 != tt.metric {
			t.Errorf("SplitGraphName(%q) = (%q, %q); want (%q, %q)", tt.name, s1, s2, tt.group, tt.metric)
		}
	}
}

func TestIsSystemMetric(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{name: "memory/used", want: true},
		{name: "memory/total", want: true},
		{name: "memory/xxx", want: false},
		{name: "xmemory/used", want: false},
		{name: "memory/usedx", want: false},
	}
	for _, tt := range tests {
		v := IsSystemMetric(tt.name)
		if v != tt.want {
			t.Errorf("IsSystemMetric(%q) = %t; want %t", tt.name, v, tt.want)
		}
	}
}

func TestGraphName(t *testing.T) {
	tests := []struct {
		name      GraphName
		matches   []string
		unmatches []string
	}{
		{
			name: GraphName("memory.avail"),
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
			name: GraphName("custom.cpu.#.user"),
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
			name: GraphName("custom.cpu.*.user"),
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
