package mackerel

import (
	"reflect"
	"testing"

	"go.opentelemetry.io/otel/api/core"
)

func TestUnmarshalHost(t *testing.T) {
	want := Meta{
		Service: ServiceMeta{
			Name:     "name",
			NS:       "ns1",
			Instance: InstanceMeta{ID: "0000-1111"},
			Version:  "a:1.1",
		},
	}
	labels := []core.KeyValue{
		core.Key("service.name").String(want.Service.Name),
		core.Key("service.namespace").String(want.Service.NS),
		core.Key("service.instance.id").String(want.Service.Instance.ID),
		core.Key("service.version").String(want.Service.Version),
	}
	var m Meta
	if err := UnmarshalHost(labels, &m); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(m, want) {
		t.Errorf("Meta = %v; want %v", m, want)
	}
}
