package mackerel

import (
	"testing"
)

func TestReplaceMetricNamePrefix(t *testing.T) {
	tests := []struct {
		prefix string
		s      string
		want   string
	}{
		{prefix: "a.b", s: "a.b.c", want: "a.b.c"},
		{prefix: "a.#", s: "a.#.c", want: "a.#.c"},
		{prefix: "a.#", s: "a.b.c", want: "a.#.c"},
		{prefix: "a.#", s: "a.b.*", want: "a.#.*"},
		{prefix: "a.#.*.x", s: "a.b.c.*", want: ""}, // fail
	}
	for _, tt := range tests {
		s, err := replaceMetricNamePrefix(tt.s, tt.prefix)
		if s != tt.want {
			t.Errorf("replaceMetricNamePrefix(%q, %q) = %q; want %q", tt.s, tt.prefix, s, tt.want)
		}
		if tt.want == "" && err == nil {
			t.Errorf("replaceMetricNamePrefix(%q, %q): want an error", tt.s, tt.prefix)
		}
	}
}

func TestGeneralizeMetricName(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{name: "memory.available", want: "memory.*"},
		{name: "memory.*", want: "memory.*"},
		{name: "memory.*.usage", want: "memory.*.usage"},
		{name: "memory.#.usage", want: "memory.#.*"},
		{name: "memory.#", want: "memory.#"},
	}
	for _, tt := range tests {
		s := generalizeMetricName(tt.name)
		if s != tt.want {
			t.Errorf("generalizeMetricName(%q) = %q; want %q", tt.name, s, tt.want)
		}
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
