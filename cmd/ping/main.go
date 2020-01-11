package main

import (
	"context"

	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/api/key"
	"go.opentelemetry.io/otel/api/metric"
)

// https://github.com/open-telemetry/opentelemetry-go/blob/master/sdk/metric/example_test.go

func main() {
	meter := global.MeterProvider().Meter("example/ping")
	key := key.New("tick")
	counter := meter.NewInt64Counter("a.counter", metric.WithKeys(key))
	labels := meter.Labels(key.String("value"))
	ctx := context.Background()
	counter.Add(ctx, 100, labels)

	v := counter.Bind(labels)
	v.Add(ctx, 20)

	m := counter.Measurement(1)
	meter.RecordBatch(ctx, labels, m)
}
