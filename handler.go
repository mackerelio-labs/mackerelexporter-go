package mackerel

import (
	"errors"
	"fmt"
	"net/http"
	"sync"
	"text/template"

	"github.com/mackerelio/mackerel-client-go"
)

type handlerClient struct {
	services map[string]*mackerel.Service
	roles    map[string]map[string]*mackerel.Role
	hosts    map[string]*mackerel.Host

	mu       sync.RWMutex
	snapshot []*mackerel.HostMetricValue
}

var _ http.Handler = &handlerClient{}

func (c *handlerClient) FindServices() ([]*mackerel.Service, error) {
	if len(c.services) == 0 {
		return nil, nil
	}
	a := make([]*mackerel.Service, 0, len(c.services))
	for _, s := range c.services {
		a = append(a, s)
	}
	return a, nil
}

func (c *handlerClient) CreateService(param *mackerel.CreateServiceParam) (*mackerel.Service, error) {
	if _, ok := c.services[param.Name]; ok {
		return nil, errors.New("the service already exists")
	}
	s := &mackerel.Service{
		Name: param.Name,
		Memo: param.Memo,
	}
	if c.services == nil {
		c.services = make(map[string]*mackerel.Service)
	}
	c.services[param.Name] = s
	return s, nil
}

func (c *handlerClient) FindRoles(serviceName string) ([]*mackerel.Role, error) {
	m := c.roles[serviceName]
	a := make([]*mackerel.Role, 0, len(m))
	for _, r := range m {
		a = append(a, r)
	}
	return a, nil
}

func (c *handlerClient) CreateRole(serviceName string, param *mackerel.CreateRoleParam) (*mackerel.Role, error) {
	m, ok := c.roles[serviceName]
	if !ok {
		m = make(map[string]*mackerel.Role)
		if c.roles == nil {
			c.roles = make(map[string]map[string]*mackerel.Role)
		}
		c.roles[serviceName] = m
	}
	if _, ok := m[param.Name]; ok {
		return nil, errors.New("the role already exists")
	}
	r := &mackerel.Role{
		Name: param.Name,
		Memo: param.Memo,
	}
	m[r.Name] = r
	return r, nil
}

func (c *handlerClient) FindHosts(param *mackerel.FindHostsParam) ([]*mackerel.Host, error) {
	// BUG(lufia): currently, FindHosts supports seraching by CustomIdentifier only.
	for _, h := range c.hosts {
		if h.CustomIdentifier == param.CustomIdentifier {
			return []*mackerel.Host{h}, nil
		}
	}
	return nil, nil
}

func (c *handlerClient) CreateHost(param *mackerel.CreateHostParam) (string, error) {
	id := fmt.Sprintf("%d", len(c.hosts)+1)
	h := &mackerel.Host{
		ID:               id,
		Name:             param.Name,
		DisplayName:      param.DisplayName,
		CustomIdentifier: param.CustomIdentifier,
		Meta:             param.Meta,
		Interfaces:       param.Interfaces,
		// Roles: meta.RoleFullnames
	}
	if c.hosts == nil {
		c.hosts = make(map[string]*mackerel.Host)
	}
	c.hosts[id] = h
	return id, nil
}

func (c *handlerClient) UpdateHost(hostID string, param *mackerel.UpdateHostParam) (string, error) {
	h, ok := c.hosts[hostID]
	if !ok {
		return "", errors.New("the host is not exist")
	}
	h.Name = param.Name
	h.DisplayName = param.DisplayName
	h.CustomIdentifier = param.CustomIdentifier
	h.Meta = param.Meta
	h.Interfaces = param.Interfaces
	// h.Roles = param.RoleFullnames
	return h.ID, nil
}

func (c *handlerClient) CreateGraphDefs(defs []*mackerel.GraphDefsParam) error {
	return nil
}

func (c *handlerClient) PostHostMetricValues(metrics []*mackerel.HostMetricValue) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.snapshot = metrics
	return nil
}

func (c *handlerClient) PostServiceMetricValues(name string, metrics []*mackerel.MetricValue) error {
	// BUG(lufia): handler-mode don't support to post the service metrics.
	return nil
}

var metricsTemplate = template.Must(template.New("metrics").Parse(`
{{- range . -}}
{{.Name}}	{{.Value}}	{{.Time}}
{{end -}}
`))

func (c *handlerClient) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c.mu.RLock()
	a := c.snapshot
	c.mu.RUnlock()

	if err := metricsTemplate.Execute(w, a); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
