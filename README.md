# mackerelexporter-go

This is the OpenTelemetry Exporter for Mackerel.

```go
import (
	"context"
	"time"

	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/api/key"
	"go.opentelemetry.io/otel/api/metric"

	"github.com/lufia/mackerelexporter"
)

func main() {
	pusher, _ := mackerel.InstallNewPipeline()
	defer pusher.Close()

	// These keys are mapped to Mackerel's attributes.
	keyHostID := key.New("host.id") // custom identifier
	keyHostName := key.New("host.name") // hostname
	keyGraphClass := key.New("mackerel.graph.class") // graph-defs's name
	keyMetricClass := key.New("mackerel.metric.class") // graph-defs's metric name

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
