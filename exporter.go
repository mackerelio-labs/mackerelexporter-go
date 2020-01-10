package mackerel

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/mackerelio/mackerel-client-go"
	"go.opencensus.io/metric/metricdata"
	"go.opencensus.io/metric/metricexport"
	"go.opencensus.io/resource/resourcekeys"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
)

const (
	UnitDimensionless = metricdata.UnitDimensionless
	UnitBytes         = metricdata.UnitBytes
	UnitMilliseconds  = metricdata.UnitMilliseconds
)

var (
	// A uniquely identifying name for the host.
	HostKeyName = tag.MustNewKey(resourcekeys.HostKeyName)

	HostKeyID = tag.MustNewKey(resourcekeys.HostKeyID)
)

// Exporter is a stats exporter that uploads data to Mackerel.
type Exporter struct {
	opts Options
	once sync.Once
	r    *metricexport.IntervalReader
	c    *mackerel.Client
}

// Options contains options for configuring the exporter.
type Options struct {
	APIKey    string
	Namespace string
}

func NewExporter(o Options) (*Exporter, error) {
	c := mackerel.NewClient(o.APIKey)
	return &Exporter{
		opts: o,
		c:    c,
	}, nil
}

// (experimental)
func makeGraphDef(v *view.View) *mackerel.GraphDefsParam {
	groupName, metricName := SplitGraphName(v.Name)

	// allows: float, integer, percentage, bytes, bytes/sec, iops
	unit := "1"
	switch v.Measure.Unit() {
	case "1":
		unit = "float"
	case "By":
		unit = "bytes"
	case "ms":
		unit = "float"
	}
	return &mackerel.GraphDefsParam{
		Name:        groupName,
		DisplayName: "",
		Unit:        unit,
		Metrics: []*mackerel.GraphDefsMetric{
			&mackerel.GraphDefsMetric{
				Name:        metricName,
				DisplayName: v.Description,
				IsStacked:   false,
			},
		},
	}
}

func (e *Exporter) ExportView(vd *view.Data) {
	fmt.Println("name:", vd.View.Name)
	for _, row := range vd.Rows {
		switch v := row.Data.(type) {
		case *view.DistributionData:
		case *view.CountData:
			fmt.Println("count:", v.Value)
		case *view.SumData:
			fmt.Println("sum:", v.Value)
		case *view.LastValueData:
			fmt.Println("last:", v.Value)
		}
	}
}

// Start starts the metric exporter.
func (e *Exporter) Start(interval time.Duration) error {
	var err error
	e.once.Do(func() {
		e.r, err = metricexport.NewIntervalReader(&metricexport.Reader{}, e)
	})
	if err != nil {
		return err
	}
	//trace.RegisterExporter(e)
	e.r.ReportingInterval = interval
	return e.r.Start()
}

func (e *Exporter) Stop() {
	//trace.UnregisterExporter(e)
	e.r.Stop()
}

func (e *Exporter) ErrLog(err error) {
	log.Println(err)
}

func (e *Exporter) ExportMetrics(ctx context.Context, data []*metricdata.Metric) error {
	a := convertToHostMetrics(data)
	if err := e.c.PostHostMetricValues(a); err != nil {
		e.ErrLog(err)
		return err
	}
	return nil
}

func convertToHostMetrics(a []*metricdata.Metric) []*mackerel.HostMetricValue {
	var r []*mackerel.HostMetricValue
	for _, p := range a {
		name := metricName(p.Descriptor)
		i := labelKeyIndex(p.Descriptor, HostKeyID.Name())
		if i < 0 {
			continue
		}
		for _, ts := range p.TimeSeries {
			if !ts.LabelValues[i].Present {
				continue
			}
			hostID := ts.LabelValues[i].Value
			a := hostMetricValues(hostID, metricValues(name, ts.Points))
			r = append(r, a...)
		}
	}
	return r
}

func metricName(d metricdata.Descriptor) string {
	s1, s2 := SplitGraphName(d.Name)
	return strings.Join([]string{s1, s2}, ".")
}

func labelKeyIndex(d metricdata.Descriptor, key string) int {
	for i, k := range d.LabelKeys {
		if k.Key == key {
			return i
		}
	}
	return -1
}

func metricValues(name string, p []metricdata.Point) []*mackerel.MetricValue {
	var a []*mackerel.MetricValue
	for _, v := range p {
		switch n := v.Value.(type) {
		case int64, float64:
			a = append(a, &mackerel.MetricValue{
				Name:  name,
				Time:  v.Time.Unix(),
				Value: n,
			})
		}
	}
	return a
}

func hostMetricValues(hostID string, a []*mackerel.MetricValue) []*mackerel.HostMetricValue {
	var r []*mackerel.HostMetricValue
	for _, v := range a {
		r = append(r, &mackerel.HostMetricValue{
			HostID:      hostID,
			MetricValue: v,
		})
	}
	return r
}

func dumpMetrics(data []*metricdata.Metric) {
	for _, v := range data {
		fmt.Println("name:", v.Descriptor.Name)
		fmt.Println("desc:", v.Descriptor.Description)
		fmt.Println("unit:", v.Descriptor.Unit)
		fmt.Println("type:", v.Descriptor.Type)
		//fmt.Println("res.type:", v.Resource.Type)
		//fmt.Println("res.labels:", v.Resource.Labels)
		for i, ts := range v.TimeSeries {
			for j, k := range v.Descriptor.LabelKeys {
				fmt.Printf("- [%d]%s=%v\n", i, k.Key, ts.LabelValues[j].Value)
			}
			for _, p := range ts.Points {
				fmt.Println(p.Time, p.Value)
				switch x := p.Value.(type) {
				case int64:
					fmt.Println("int:", x)
				case float64:
					fmt.Println("float:", x)
				case *metricdata.Distribution:
					fmt.Println("distribution:", x)
				case *metricdata.Summary:
					fmt.Println("summary:", x)
				}
			}
		}
		fmt.Println("----------------")
	}
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

func (e *Exporter) RegisterHost(h *Host) (string, error) {
	id, err := e.lookupHost(h)
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

func (e *Exporter) lookupHost(h *Host) (string, error) {
	if h.CustomIdentifier != "" {
		a, err := e.c.FindHosts(&mackerel.FindHostsParam{
			CustomIdentifier: h.CustomIdentifier,
		})
		if err != nil {
			return "", err
		}
		if len(a) > 0 {
			return a[0].ID, nil
		}
	}
	return "", nil // not found
}
