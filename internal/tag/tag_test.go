package tag

import (
	"reflect"
	"testing"

	"go.opentelemetry.io/otel/api/core"
)

func TestUnmarshalTags(t *testing.T) {
	want := Resource{
		Service: Service{
			Name:     "name",
			NS:       "ns1",
			Instance: Instance{ID: "0000-1111"},
			Version:  "a:1.1",
		},
	}
	labels := []core.KeyValue{
		core.Key("service.name").String(want.Service.Name),
		core.Key("service.namespace").String(want.Service.NS),
		core.Key("service.instance.id").String(want.Service.Instance.ID),
		core.Key("service.version").String(want.Service.Version),
	}
	var m Resource
	if err := UnmarshalTags(labels, &m); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(m, want) {
		t.Errorf("Resource = %v; want %v", m, want)
	}
}

func TestUnmarshalTagsInterface(t *testing.T) {
	want := Resource{
		Service: Service{
			Name:     "name",
			NS:       "ns1",
			Instance: Instance{ID: "0000-1111"},
			Version:  "a:1.1",
		},
	}
	labels := []core.KeyValue{
		core.Key("service.name").String(want.Service.Name),
		core.Key("service.namespace").String(want.Service.NS),
		core.Key("service.instance.id").String(want.Service.Instance.ID),
		core.Key("service.version").String(want.Service.Version),
	}
	var v interface{}
	if err := UnmarshalTags(labels, &v); err != nil {
		t.Fatal(err)
	}
	if s, ok := lookupInterfaceMap(v, "service", "name").(string); !ok || s != want.Service.Name {
		t.Errorf("service.name = %s; want %s", s, want.Service.Name)
	}
	if s, ok := lookupInterfaceMap(v, "service", "instance", "id").(string); !ok || s != want.Service.Instance.ID {
		t.Errorf("service.instance.id = %s; want %s", s, want.Service.Instance.ID)
	}
}

func lookupInterfaceMap(v interface{}, keys ...string) interface{} {
	for _, key := range keys {
		m, ok := v.(map[string]interface{})
		if !ok {
			return nil
		}
		p, ok := m[key]
		if !ok {
			return nil
		}
		v = p
	}
	return v
}

func TestCustomIdentifier(t *testing.T) {
	tests := []struct {
		r    Resource
		want string
	}{
		{
			r: Resource{
				Host: Host{ID: "host_id"},
			},
			want: "host_id",
		},
		{
			r: Resource{
				Service: Service{
					NS:       "ns",
					Name:     "name",
					Instance: Instance{ID: "i-xxx"},
				},
			},
			want: "ns/name/i-xxx",
		},
		{
			r: Resource{
				Service: Service{
					NS:   "ns",
					Name: "name",
				},
			},
			want: "",
		},
	}
	for _, tt := range tests {
		s := tt.r.CustomIdentifier()
		if s != tt.want {
			t.Errorf("CustomIdentifier = %q; want %q", s, tt.want)
		}
	}
}
