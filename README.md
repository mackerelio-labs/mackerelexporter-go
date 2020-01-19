# mackerelexporter-go

This is the OpenTelemetry Exporter for Mackerel.

[![GoDoc][godoc-image]][godoc-url]
[![Actions Status][actions-image]][actions-url]

## Hosts
TODO

## Graph Definitions
TODO

## Example

```go
import (
	"context"
	"os"
	"time"

	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/api/key"
	"go.opentelemetry.io/otel/api/metric"

	"github.com/lufia/mackerelexporter"
)

var (
	// These keys are mapped to Mackerel's attributes.
	keyHostID      = key.New("host.id")               // custom identifier
	keyHostName    = key.New("host.name")             // hostname
	keyGraphClass  = key.New("mackerel.graph.class")  // graph-def's name
	keyMetricClass = key.New("mackerel.metric.class") // graph-def's metric name
)

func main() {
	apiKey := os.Getenv("MACKEREL_APIKEY")
	pusher, _ := mackerel.InstallNewPipeline(mackerel.WithAPIKey(apiKey))
	defer pusher.Stop()

	meter := global.MeterProvider().Meter("example")
	firestoreRead := meter.NewInt64Counter("storage.firestore.read", metric.WithKeys(
		keyHostID, keyHostName, keyGraphClass, keyMetricClass,
	))
	labels := meter.Labels(
		keyHostID.String("10-1-2-241"),
		keyHostName.String("localhost"),
		keyGraphClass.String("storage.#"),
		keyMetricClass.String("storage.#.*"),
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

[godoc-image]: https://godoc.org/github.com/lufia/mackerelexporter-go?status.svg
[godoc-url]: https://godoc.org/github.com/lufia/mackerelexporter-go
[actions-image]: https://github.com/lufia/mackerelexporter-go/workflows/ci/badge.svg
[actions-url]: https://github.com/lufia/mackerelexporter-go/actions
