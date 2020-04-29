package metricname

import (
	"testing"
)

func TestSanitize(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{name: "abc.def", want: "abc.def"},
		{name: "custom.#.*.t-x_a", want: "custom.#.*.t-x_a"},
		{name: "aaa.$!.bb", want: "aaa.__.bb"},
	}
	for _, tt := range tests {
		s := Sanitize(tt.name)
		if s != tt.want {
			t.Errorf("Sanitize(%q) = %q; want %q", tt.name, s, tt.want)
		}
	}
}

func TestMatch(t *testing.T) {
	tests := []struct {
		name      string
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
				if ok := Match(s, tt.name); !ok {
					t.Errorf("Match(%q, %q) = %t", s, tt.name, ok)
				}
			}
		})
		t.Run("unmatches", func(t *testing.T) {
			for _, s := range tt.unmatches {
				if ok := Match(s, tt.name); ok {
					t.Errorf("Match(%q, %q) = %t", s, tt.name, ok)
				}
			}
		})
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
