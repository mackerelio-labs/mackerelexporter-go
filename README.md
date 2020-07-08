# mackerelexporter-go

This is OpenTelemetry Meter Exporter for [Mackerel](https://mackerel.io/).

[![GoDev][godev-image]][godev-url]
[![Actions Status][actions-image]][actions-url]

## Installation

If you use Go Modules, then you just import OpenTelemetry API and the exporter.  Otherwise you can install manually.

```console
$ go get -u go.opentelemetry.io/otel
$ go get -u github.com/mackerelio-labs/mackerelexporter-go
```

After installation, you can start to write a program. There is the example at the Example section of this document.

## Description

### Hosts
Mackerel manages hosts database in the organization. Official mackerel-agent installed to the host, such as compute instance, bare metal or local machine collects host informations from resources provided by operating system. The exporter won't access external resources. Instead host informations will be generated from labels attached to the metrics.

For example, the host identifier that is unique in the organization refers the label `host.id`. Similarly the host name refers the label `host.name`. The exporter handles metrics attached these labels as host metric in Mackerel.

### Services
Like as hosts, both Service and Role are made from labels. Label `service.namespace` is mapped to Service, and the label `service.name` is mapped Role.

### Where post are metrics?
The metrics with a label below will post as Host Metric.

- `host.id`

Also the metrics with these all labels below will post as Host Metric.

- `service.namespace`
- `service.name`
- `service.instance.id`

The metric with a labels below will post as Service Metric.

- `service.namespace`

### Graph Definitions
The exporter will create the Graph Definition on Mackerel if needed. Most cases it creates automatically based from recorded metric name. However you might think to want to customize the graph by wildcards in the graph name. In this case you can configure the exporter to use pre-defined graph name with *WithHints()* option.

The hint is the name composed by deleted only the end of the metric name, and it can be replaced each elements by wildcard, `#` or `*`. For example the hint `http.handlers.#` is valid for the metric `http.handlers.index.count`.

Special case. The exporter will append *.min*, *.max* and *.percentile_xx* implicitly to the end of the name for *measure* metric in OpenTelemetry. Thus count of elements of the hint will be same as the recorded metric name.

## The push/pull mode

If you give *InstallNewPipeline* a valid API key with *WithAPIKey* option, the exporter runs as the push mode. In this mode, the exporter sends host- and service-metrics to Mackerl automatically. Otherwise the exporter runs as the pull mode. The pull mode dont' send any metrics. Instead, *InstallNewPipeline* returns a handler function for *net/http*. In pull mode, the handler function responds host metrics to the HTTP client, and it don't include any service metrics.

## Example

```go
import (
	"context"
	"log"
	"net/http"
	"os"
	"runtime"
	"time"

	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/api/kv"
	"go.opentelemetry.io/otel/api/metric"
	"go.opentelemetry.io/otel/api/unit"

	"github.com/mackerelio-labs/mackerelexporter-go"
)

var (
	hints = []string{
		"storage.#",
	}
)

func main() {
	apiKey := os.Getenv("MACKEREL_APIKEY")
	pusher, handler, err := mackerel.InstallNewPipeline(
		mackerel.WithAPIKey(apiKey),
		mackerel.WithHints(hints),
		mackerel.WithResource([]kv.KeyValue{
			mackerel.KeyHostID.String("10-1-2-241"),
			mackerel.KeyHostName.String("localhost"),
			mackerel.KeyServiceNS.String("example"), // service name
			mackerel.KeyServiceName.String("app"),   // role name
		}),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer pusher.Stop()
	if handler != nil {
		http.HandlerFunc("/metrics", handler)
		go http.ListenAndServe(":8080", nil)
	}

	meter := global.MeterProvider().Meter("example")
	meterMust := metric.Must(meter)
	firestoreRead := meterMust.NewInt64Counter("storage.firestore.read")
	var (
		memAllocs      metric.Int64ValueObserver
		memTotalAllocs metric.Int64ValueObserver
	)
	memStats := meterMust.NewBatchObserver(func(ctx context.Context, result metric.BatchObserverResult) {
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		result.Observe(nil,
			memAllocs.Observation(int64(m.Alloc)),
			memTotalAllocs.Observation(int64(m.TotalAlloc)),
		)
	})
	memAllocs = memStats.NewInt64ValueObserver("runtime.memory.alloc", metric.WithUnit(unit.Bytes))
	memTotalAllocs = memStats.NewInt64ValueObserver("runtime.memory.total_alloc", metric.WithUnit(unit.Bytes))

	ctx := context.Background()
	additionalLabels := []kv.KeyValue{
		mackerel.KeyServiceVersion("v1.0"),
	}
	firestoreRead.Add(ctx, 100, additionalLabels)

	v := firestoreRead.Bind(additionalLabels)
	v.Add(ctx, 20)

	m := firestoreRead.Measurement(1)
	meter.RecordBatch(ctx, additionalLabels, m)
	time.Sleep(2 * time.Minute)
}
```

[godev-image]: https://img.shields.io/badge/go.dev-reference-007d9c?logo=go&logoColor=white&style=flat-square
[godev-url]: https://pkg.go.dev/github.com/mackerelio-labs/mackerelexporter-go
[actions-image]: https://github.com/mackerelio-labs/mackerelexporter-go/workflows/ci/badge.svg
[actions-url]: https://github.com/mackerelio-labs/mackerelexporter-go/actions
