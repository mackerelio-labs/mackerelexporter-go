package mackerel

import (
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
	var h Host
	for k, s := range meta {
		if !kv.Key.Defined() {
			continue
		}
		keys := strings.Split(string(kv.Key), ".")
		unmarshalHost(keys, kv.Value.Emit(), data)
	}
	reflect
}

func unmarshalHost(keys []string, value string, data interface{}) error {
	v := reflect.ValueOf(data)
	if v.Type().Kind() {
	}
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
