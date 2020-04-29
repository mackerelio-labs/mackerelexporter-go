# mackerelexporter-go

This is OpenTelemetry Meter Exporter for [Mackerel](https://mackerel.io/).

[![GoDev][godev-image]][godev-url]
[![Actions Status][actions-image]][actions-url]

## Installation

If you use Go Modules, then you just import OpenTelemetry API and the exporter.  Otherwise you can install manually.

```console
$ go get -u go.opentelemetry.io/otel
$ go get -u github.com/lufia/mackerelexporter-go
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

## Example

```go
import (
	"context"
	"os"
	"time"

	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/api/key"
	"go.opentelemetry.io/otel/api/metric"

	"github.com/lufia/mackerelexporter-go"
)

var (
	// These keys are mapped to Mackerel's attributes.
	keyHostID      = key.New("host.id")               // custom identifier
	keyHostName    = key.New("host.name")             // hostname

	hints = []string{
		"storage.#",
	}
)

func main() {
	apiKey := os.Getenv("MACKEREL_APIKEY")
	pusher, _ := mackerel.InstallNewPipeline(
		mackerel.WithAPIKey(apiKey),
		mackerel.WithHints(hints),
	)
	defer pusher.Stop()

	meter := global.MeterProvider().Meter("example")
	firestoreRead := meter.NewInt64Counter("storage.firestore.read", metric.WithKeys(
		keyHostID, keyHostName, keyGraphClass, keyMetricClass,
	))
	labels := meter.Labels(
		keyHostID.String("10-1-2-241"),
		keyHostName.String("localhost"),
	)
	ctx := context.Background()
	firestoreRead.Add(ctx, 100, labels)

	v := firestoreRead.Bind(labels)
	v.Add(ctx, 20)

	m := firestoreRead.Measurement(1)
	meter.RecordBatch(ctx, labels, m)
	time.Sleep(2 * time.Minute)
}
```

[godev-image]: https://img.shields.io/badge/go.dev-reference-007d9c?logo=go&logoColor=white&style=flat-square
[godev-url]: https://pkg.go.dev/github.com/lufia/mackerelexporter-go
[actions-image]: https://github.com/lufia/mackerelexporter-go/workflows/ci/badge.svg
[actions-url]: https://github.com/lufia/mackerelexporter-go/actions
