package mackerel

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"strings"

	"github.com/mackerelio/mackerel-client-go"
	"go.opentelemetry.io/otel/api/core"
)

type Resource struct {
	Service ServiceResource  `resource:"service"`
	Host    HostResource     `resource:"host"`
	Cloud   CloudResource    `resource:"cloud"`
	Mackrel MackerelResource `resource:"mackerel"`
}

type ServiceResource struct {
	Name     string           `resource:"name"`
	NS       string           `resource:"namespace"`
	Instance InstanceResource `resource:"instance"`
	Version  string           `resource:"version"`
}

type InstanceResource struct {
	ID string `resource:"id"`
}

type HostResource struct {
	ID   string `resource:"id"`
	Name string `resource:"name"`
}

type CloudResource struct {
	Provider string `resource:"provider"`
}

// MackerelResource represents Mackerel specific resources.
type MackerelResource struct {
	Metric struct {
		Class string `resource:"class"`
	} `resource:"metric"`
}

// UnmarshalLabels marshals ...
func UnmarshalLabels(meta []core.KeyValue, data interface{}) error {
	v := reflect.ValueOf(data)
	for _, kv := range meta {
		if !kv.Key.Defined() {
			continue
		}
		keys := strings.Split(string(kv.Key), ".")
		if err := unmarshalLabels(keys, kv.Value, v); err != nil {
			return err
		}
	}
	return nil
}

func unmarshalLabels(keys []string, value core.Value, v reflect.Value) error {
	switch kind := v.Type().Kind(); kind {
	case reflect.Ptr:
		return unmarshalLabels(keys, value, reflect.Indirect(v))
	case reflect.Struct:
		if len(keys) == 0 {
			return errors.New("missing")
		}
		fields := collectFields(v)
		f, ok := fields[keys[0]]
		if !ok {
			return nil // ignore this field
		}
		return unmarshalLabels(keys[1:], value, f)
	case reflect.Interface:
		if v.IsNil() {
			v.Set(reflect.ValueOf(map[string]interface{}{}))
		}
		return unmarshalLabels(keys, value, v.Elem())
	case reflect.Map:
		if len(keys) == 0 {
			return errors.New("cannot map a value to a map")
		}
		key := reflect.ValueOf(keys[0])
		if len(keys) == 1 {
			v.SetMapIndex(key, reflect.ValueOf(value.Emit()))
			return nil
		}
		p := v.MapIndex(key)
		if !p.IsValid() {
			p = reflect.ValueOf(map[string]interface{}{})
			v.SetMapIndex(key, p)
		}
		if err := unmarshalLabels(keys[1:], value, p); err != nil {
			return err
		}
		return nil
	case reflect.Bool:
		if len(keys) != 0 {
			return errors.New("overflow")
		}
		if v.CanSet() {
			v.SetBool(value.AsBool())
		}
		return nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if len(keys) != 0 {
			return errors.New("overflow")
		}
		if v.CanSet() {
			v.SetInt(value.AsInt64())
		}
		return nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if len(keys) != 0 {
			return errors.New("overflow")
		}
		if v.CanSet() {
			v.SetUint(value.AsUint64())
		}
		return nil
	case reflect.Float32, reflect.Float64:
		if len(keys) != 0 {
			return errors.New("overflow")
		}
		if v.CanSet() {
			v.SetFloat(value.AsFloat64())
		}
		return nil
	case reflect.String:
		if len(keys) != 0 {
			return errors.New("overflow")
		}
		if v.CanSet() {
			v.SetString(value.AsString())
		}
		return nil
	default:
		//  Uintptr
		return fmt.Errorf("unsupported type: %v", kind)
	}
}

// collectFields returns a map pointed to fields by the `meta` tag.
func collectFields(v reflect.Value) map[string]reflect.Value {
	a := make(map[string]reflect.Value)
	t := v.Type()
	n := v.NumField()
	for i := 0; i < n; i++ {
		f := t.Field(i)
		name := f.Tag.Get("resource")
		if name == "" {
			name = f.Name
		}
		a[name] = v.Field(i)
	}
	return a
}

type Host struct {
	Name             string
	CustomIdentifier string
	Meta             map[string]interface{}
}

func makeHost(res *Resource) *Host {
	var h Host
	h.Name = res.Host.Name
	if h.Name == "" {
		h.Name = res.Service.Instance.ID
	}
	if h.Name == "" {
		if name, err := os.Hostname(); err == nil {
			h.Name = name
		}
	}
	h.CustomIdentifier = customIdentifier(res)
	// TODO(lufia): Should we set any values to h.Meta?
	return &h
}

func customIdentifier(res *Resource) string {
	// TODO(lufia): This may change to equal to mackerel-agent.
	a := make([]string, 0, 3)
	if s := res.Service.NS; s != "" {
		a = append(a, s)
	}
	if s := res.Service.Name; s != "" {
		a = append(a, s)
	}
	if s := res.Service.Instance.ID; s != "" {
		a = append(a, s)
	}
	return strings.Join(a, ".")
}

// UpsertHost update or insert the host with res.
func (e *Exporter) UpsertHost(res *Resource) (hostID string, err error) {
	h := makeHost(res)
	param := mackerel.CreateHostParam{
		Name:             h.Name,
		CustomIdentifier: h.CustomIdentifier,
	}
	if res.Cloud.Provider != "" {
		param.Meta = mackerel.HostMeta{
			Cloud: &mackerel.Cloud{
				Provider: res.Cloud.Provider,
			},
		}
	}

	id := res.Host.ID
	if id == "" {
		id, err = e.lookupHostID(h.CustomIdentifier)
		if err != nil {
			return
		}
	}
	if id != "" {
		// TODO(lufia): we should update a host
		return id, nil // The host was already registered
	}
	return e.c.CreateHost(&param)
}

func (e *Exporter) lookupHostID(customIdentifier string) (string, error) {
	if customIdentifier == "" {
		return "", errors.New("customIdentifier must be specified")
	}
	a, err := e.c.FindHosts(&mackerel.FindHostsParam{
		CustomIdentifier: customIdentifier,
	})
	if err != nil {
		return "", err
	}
	if len(a) == 0 {
		return "", nil
	}
	return a[0].ID, nil
}
