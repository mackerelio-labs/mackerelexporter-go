package mackerel

import (
	"errors"

	"github.com/lufia/mackerelexporter-go/internal/tag"
	"github.com/mackerelio/mackerel-client-go"
)

func (e *Exporter) registerService(name string) error {
	if _, ok := e.serviceRoles[name]; ok {
		return nil
	}
	a, err := e.c.FindServices()
	if err != nil {
		return err
	}
	for _, s := range a {
		if s.Name == name {
			e.serviceRoles[name] = make(map[string]struct{})
			return nil
		}
	}

	param := mackerel.CreateServiceParam{
		Name: name,
	}
	if _, err = e.c.CreateService(&param); err != nil {
		return err
	}
	e.serviceRoles[name] = make(map[string]struct{})
	return nil
}

func (e *Exporter) registerServiceRole(s, role string) error {
	if err := e.registerService(s); err != nil {
		return err
	}
	if _, ok := e.serviceRoles[s][role]; ok {
		return nil
	}
	a, err := e.c.FindRoles(s)
	if err != nil {
		return err
	}
	for _, r := range a {
		if r.Name == role {
			e.serviceRoles[s][role] = struct{}{}
			return nil
		}
	}

	param := mackerel.CreateRoleParam{
		Name: role,
	}
	if _, err := e.c.CreateRole(s, &param); err != nil {
		return err
	}
	e.serviceRoles[s][role] = struct{}{}
	return nil
}

// upsertHost update or insert the host with r.
func (e *Exporter) upsertHost(r *tag.Resource) (string, error) {
	param := mackerel.CreateHostParam{
		Name:             r.Hostname(),
		CustomIdentifier: r.CustomIdentifier(),
	}
	if roleFullname := r.RoleFullname(); roleFullname != "" {
		s := r.ServiceName()
		role := r.RoleName()
		if err := e.registerServiceRole(s, role); err != nil {
			return "", err
		}
		param.RoleFullnames = []string{roleFullname}
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
