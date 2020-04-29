package tag

import (
	"fmt"
	"os"
	"reflect"
	"strings"

	"go.opentelemetry.io/otel/api/core"
	"go.opentelemetry.io/otel/sdk/resource/resourcekeys"
)

const (
	resourceNameSep = "."
)

// These keys are handled for creating hosts, graph-defs, or metrics.
var (
	// see https://github.com/open-telemetry/opentelemetry-specification/blob/master/specification/resource/semantic_conventions/README.md
	keyServiceNS         = core.Key(resourcekeys.ServiceKeyNamespace)
	keyServiceName       = core.Key(resourcekeys.ServiceKeyName)
	keyServiceInstanceID = core.Key(resourcekeys.ServiceKeyInstanceID)
	keyServiceVersion    = core.Key(resourcekeys.ServiceKeyVersion)
	keyHostID            = core.Key(resourcekeys.HostKeyID)
	keyHostName          = core.Key(resourcekeys.HostKeyName)
	keyCloudProvider     = core.Key(resourcekeys.CloudKeyProvider)
)

// Resource represents a resource constructed with labels.
type Resource struct {
	Service Service `resource:"service"`
	Host    Host    `resource:"host"`
	Cloud   Cloud   `resource:"cloud"`
}

// Service represents the standard service attributes.
type Service struct {
	Name     string   `resource:"name"`
	NS       string   `resource:"namespace"`
	Instance Instance `resource:"instance"`
	Version  string   `resource:"version"`
}

// Instance represents the standard instance attributes.
type Instance struct {
	ID string `resource:"id"`
}

// Host represents the standard host attributes.
type Host struct {
	ID   string `resource:"id"`
	Name string `resource:"name"`
}

// Cloud represents the standard cloud attributes.
type Cloud struct {
	Provider string `resource:"provider"`
}

// Hostname returns a proper hostname.
func (r *Resource) Hostname() string {
	if r.Host.Name != "" {
		return r.Host.Name
	}
	if s := r.nameFromService("-"); s != "" {
		return s
	}
	if s, _ := os.Hostname(); s != "" {
		return s
	}
	return "localhost"
}

// CustomIdentifier returns a proper customIdentifier for the host.
func (r *Resource) CustomIdentifier() string {
	if r.Host.ID != "" {
		return r.Host.ID
	}
	return r.nameFromService("/")
}

func (r *Resource) nameFromService(sep string) string {
	if r.Service.Instance.ID == "" {
		return ""
	}
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
	return strings.Join(a, sep)
}

// ServiceName returns a service name.
func (r *Resource) ServiceName() string {
	return r.Service.NS
}

// RoleName returns a role name.
func (r *Resource) RoleName() string {
	return r.Service.Name
}

// RoleFullname returns a full qualified role name.
func (r *Resource) RoleFullname() string {
	if r.Service.NS == "" || r.Service.Name == "" {
		return ""
	}
	return r.Service.NS + ":" + r.Service.Name
}

// UnmarshalTags parses labels and store the result into v.
func UnmarshalTags(tags []core.KeyValue, v interface{}) error {
	p := reflect.ValueOf(v)
	for _, tag := range tags {
		if !tag.Key.Defined() {
			continue
		}
		name := string(tag.Key)
		keys := strings.Split(name, resourceNameSep)
		if err := unmarshalTags("<v>", keys, tag.Value, p); err != nil {
			return fmt.Errorf("cannot assign %s: %w", name, err)
		}
	}
	return nil
}

// name must mean v
func unmarshalTags(name string, keys []string, value core.Value, v reflect.Value) error {
	switch kind := v.Type().Kind(); kind {
	case reflect.Ptr:
		return unmarshalTags(name, keys, value, reflect.Indirect(v))
	case reflect.Struct:
		if len(keys) == 0 {
			return fmt.Errorf("%s is %v", name, kind)
		}
		fields := collectFields(v)
		f, ok := fields[keys[0]]
		if !ok {
			return nil // ignore this field
		}
		return unmarshalTags(keys[0], keys[1:], value, f)
	case reflect.Interface:
		if v.IsNil() {
			v.Set(reflect.ValueOf(map[string]interface{}{}))
		}
		return unmarshalTags(name, keys, value, v.Elem())
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
		if err := unmarshalTags(keys[0], keys[1:], value, p); err != nil {
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
