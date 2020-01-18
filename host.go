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

const (
	resourceNameSep = "."
)

var (
	// see https://github.com/open-telemetry/opentelemetry-specification/blob/master/specification/data-resource-semantic-conventions.md
	keyServiceNS         = core.Key("service.namespace")
	keyServiceName       = core.Key("service.name")
	keyServiceInstanceID = core.Key("service.instance.id")
	keyServiceVersion    = core.Key("service.version")
	keyHostID            = core.Key("host.id")
	keyHostName          = core.Key("host.name")
	keyCloudProvider     = core.Key("cloud.provider")

	// for graph-def
	keyGraphClass  = core.Key("mackerel.graph.class")
	keyMetricClass = core.Key("mackerel.metric.class")
)

// Resource represents a resource constructed with labels.
type Resource struct {
	Service  ServiceResource  `resource:"service"`
	Host     HostResource     `resource:"host"`
	Cloud    CloudResource    `resource:"cloud"`
	Mackerel MackerelResource `resource:"mackerel"`
}

// ServiceResource represents the standard service attributes.
type ServiceResource struct {
	Name     string           `resource:"name"`
	NS       string           `resource:"namespace"`
	Instance InstanceResource `resource:"instance"`
	Version  string           `resource:"version"`
}

// InstanceResource represents the standard instance attributes.
type InstanceResource struct {
	ID string `resource:"id"`
}

// HostResource represents the standard host attributes.
type HostResource struct {
	ID   string `resource:"id"`
	Name string `resource:"name"`
}

// CloudResource represents the standard cloud attributes.
type CloudResource struct {
	Provider string `resource:"provider"`
}

// MackerelResource represents Mackerel specific resources.
type MackerelResource struct {
	Metric struct {
		Class string `resource:"class"`
	} `resource:"metric"`
	Graph struct {
		Class string `resource:"class"`
	} `resource:"graph"`
}

// Hostname returns a proper hostname.
func (r *Resource) Hostname() string {
	if r.Host.Name != "" {
		return r.Host.Name
	}
	if r.Service.Instance.ID != "" {
		prefix := ""
		if r.Service.Name != "" {
			prefix = r.Service.Name + "-"
		}
		return prefix + r.Service.Instance.ID
	}
	if s, err := os.Hostname(); err == nil {
		return s
	}
	return ""
}

// CustomIdentifier returns a proper customIdentifier for the host.
func (r *Resource) CustomIdentifier() string {
	if r.Host.ID != "" {
		return r.Host.ID
	}

	// TODO(lufia): This may change to equal to mackerel-agent.
	a := make([]string, 0, 3)
	if s := r.Service.NS; s != "" {
		a = append(a, s)
	}
	if s := r.Service.Name; s != "" {
		a = append(a, s)
	}
	if s := r.Service.Instance.ID; s != "" {
		a = append(a, s)
	}
	return strings.Join(a, resourceNameSep)
}

// UnmarshalLabels parses labels and store the result into v.
func UnmarshalLabels(labels []core.KeyValue, v interface{}) error {
	p := reflect.ValueOf(v)
	for _, kv := range labels {
		if !kv.Key.Defined() {
			continue
		}
		name := string(kv.Key)
		keys := strings.Split(name, resourceNameSep)
		if err := unmarshalLabels("<v>", keys, kv.Value, p); err != nil {
			return fmt.Errorf("cannot assign %s: %w", name, err)
		}
	}
	return nil
}

// name must mean v
func unmarshalLabels(name string, keys []string, value core.Value, v reflect.Value) error {
	switch kind := v.Type().Kind(); kind {
	case reflect.Ptr:
		return unmarshalLabels(name, keys, value, reflect.Indirect(v))
	case reflect.Struct:
		if len(keys) == 0 {
			return fmt.Errorf("%s is %v", name, kind)
		}
		fields := collectFields(v)
		f, ok := fields[keys[0]]
		if !ok {
			return nil // ignore this field
		}
		return unmarshalLabels(keys[0], keys[1:], value, f)
	case reflect.Interface:
		if v.IsNil() {
			v.Set(reflect.ValueOf(map[string]interface{}{}))
		}
		return unmarshalLabels(name, keys, value, v.Elem())
	case reflect.Map:
		if len(keys) == 0 {
			return fmt.Errorf("%s is %v", name, kind)
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
		if err := unmarshalLabels(keys[0], keys[1:], value, p); err != nil {
			return err
		}
		return nil
	case reflect.Bool:
		if len(keys) != 0 {
			return fmt.Errorf("%s is %v", keys[0], kind)
		}
		if v.CanSet() {
			v.SetBool(value.AsBool())
		}
		return nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if len(keys) != 0 {
			return fmt.Errorf("%s is %v", keys[0], kind)
		}
		if v.CanSet() {
			v.SetInt(value.AsInt64())
		}
		return nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if len(keys) != 0 {
			return fmt.Errorf("%s is %v", keys[0], kind)
		}
		if v.CanSet() {
			v.SetUint(value.AsUint64())
		}
		return nil
	case reflect.Float32, reflect.Float64:
		if len(keys) != 0 {
			return fmt.Errorf("%s is %v", keys[0], kind)
		}
		if v.CanSet() {
			v.SetFloat(value.AsFloat64())
		}
		return nil
	case reflect.String:
		if len(keys) != 0 {
			return fmt.Errorf("%s is %v", keys[0], kind)
		}
		if v.CanSet() {
			v.SetString(value.AsString())
		}
		return nil
	default:
		//  Uintptr
		return fmt.Errorf("%s: unsupported type: %v", name, kind)
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

// UpsertHost update or insert the host with r.
func (e *Exporter) UpsertHost(r *Resource) (string, error) {
	// TODO(lufia): We would require to redesign whether using mackerel-client-go or not.
	param := mackerel.CreateHostParam{
		Name:             r.Hostname(),
		CustomIdentifier: r.CustomIdentifier(),
	}
	if r.Cloud.Provider != "" {
		param.Meta = mackerel.HostMeta{
			Cloud: &mackerel.Cloud{
				Provider: r.Cloud.Provider,
			},
		}
	}

	hostID, err := e.lookupHostID(param.CustomIdentifier)
	if err != nil {
		return "", err
	}
	if hostID == "" {
		return e.c.CreateHost(&param)
	}
	return e.c.UpdateHost(hostID, (*mackerel.UpdateHostParam)(&param))
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
