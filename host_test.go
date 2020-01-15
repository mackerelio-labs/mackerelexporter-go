package mackerel

import (
	"testing"

	"go.opentelemetry.io/otel/api/core"
)

func TestUnmarshalHost(t *testing.T) {
	labels := []core.KeyValue{
		core.Key("service.name").String("test"),
	}
	var m Meta
	if err := UnmarshalHost(labels, &m); err != nil {
		t.Fatal(err)
	}
	if want := "test"; m.Service.Name != want {
		t.Errorf("Service.Name = %s; want %s", m.Service.Name, want)
	}
}
