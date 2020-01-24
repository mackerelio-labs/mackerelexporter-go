package mackerel

import (
	"errors"

	"github.com/lufia/mackerelexporter-go/internal/resource"
	"github.com/mackerelio/mackerel-client-go"
)

// upsertHost update or insert the host with r.
func (e *Exporter) upsertHost(r *resource.Resource) (string, error) {
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

func (e *Exporter) createService(name string) error {
	a, err := e.c.FindServices()
	if err != nil {
		return err
	}
	for _, s := range a {
		if s.Name == name {
			return nil
		}
	}

	param := mackerel.CreateServiceParam{
		Name: name,
	}
	_, err = e.c.CreateService(&param)
	return err
}
