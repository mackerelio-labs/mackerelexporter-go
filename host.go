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

type Meta struct {
	Service ServiceMeta `resource:"service"`
}

type ServiceMeta struct {
	Name     string       `resource:"name"`
	NS       string       `resource:"namespace"`
	Instance InstanceMeta `resource:"instance"`
	Version  string       `resource:"version"`
}

type InstanceMeta struct {
	ID string `resource:"id"`
}

type Host struct {
	Name             string
	CustomIdentifier string
	Meta             HostMeta

	//Roles            Roles
	//Interfaces       []Interface
}

type HostMeta struct {
	AgentVersion string
	AgentName    string
	CPUName      string
	CPUMHz       int

	//BlockDevice   BlockDevice
	//Filesystem    FileSystem
	//Memory        Memory
	//Cloud         *Cloud
}

func UnmarshalHost(meta []core.KeyValue, data interface{}) error {
	v := reflect.ValueOf(data)
	for _, kv := range meta {
		if !kv.Key.Defined() {
			continue
		}
		keys := strings.Split(string(kv.Key), ".")
		if err := unmarshalHost(keys, kv.Value, v); err != nil {
			return err
		}
	}
	return nil
}

func unmarshalHost(keys []string, value core.Value, v reflect.Value) error {
	switch kind := v.Type().Kind(); kind {
	case reflect.Ptr:
		return unmarshalHost(keys, value, reflect.Indirect(v))
	case reflect.Struct:
		if len(keys) == 0 {
			return errors.New("missing")
		}
		fields := collectFields(v)
		f, ok := fields[keys[0]]
		if !ok {
			return nil // ignore this field
		}
		return unmarshalHost(keys[1:], value, f)
	case reflect.Map:
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

func makeHost(meta map[core.Key]string) *Host {
	var h Host
	if name, err := os.Hostname(); err == nil {
		h.Name = name
	}
	if name, ok := meta[keyHostName]; ok {
		h.Name = name
	}

	if id, ok := meta[keyServiceInstanceID]; ok {
		h.CustomIdentifier = id
	}

	h.Meta.AgentName = "mackerel-exporter (ot)"
	h.Meta.AgentVersion = "0.1"
	return &h
}

func customIdentifier(meta map[core.Key]string) string {
	a := make([]string, 0, 3)
	if s, ok := meta[keyServiceNS]; ok {
		a = append(a, s)
	}
	s, ok := meta[keyServiceName]
	if !ok {
		return "" // wrong; service.name is required
	}
	a = append(a, s)
	s, ok = meta[keyServiceInstanceID]
	if !ok {
		return "" // wrong; service.instance.id is required
	}
	a = append(a, s)
	return strings.Join(a, ".")
}

func (e *Exporter) upsertHost(h *Host) (string, error) {
	id, err := e.lookupHostID(h.CustomIdentifier)
	if err != nil {
		return "", err
	}
	if id != "" {
		// TODO(lufia): we should update a host
		return id, nil // The host was already registered
	}

	cpu0 := map[string]interface{}{
		"model_name": h.Meta.CPUName,
		"mhz":        h.Meta.CPUMHz,
	}
	param := mackerel.CreateHostParam{
		Name:             h.Name,
		CustomIdentifier: h.CustomIdentifier,
		Meta: mackerel.HostMeta{
			AgentVersion: h.Meta.AgentVersion,
			AgentName:    h.Meta.AgentName,
			CPU:          mackerel.CPU{cpu0},
			Kernel: map[string]string{
				"os":      "Plan 9",
				"release": "4e",
				"version": "2000",
			},
		},
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
