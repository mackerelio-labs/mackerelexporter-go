package main

import (
	"context"
	"log"
	"time"

	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/api/key"
	"go.opentelemetry.io/otel/api/metric"

	"github.com/lufia/mackerelexporter"
)

// https://github.com/open-telemetry/opentelemetry-go/blob/master/sdk/metric/example_test.go

func main() {
	log.SetFlags(0)
	pusher, err := mackerel.InstallNewPipeline()
	if err != nil {
		log.Fatal(err)
	}
	defer pusher.Close()

	meter := global.MeterProvider().Meter("example/ping")
	key := key.New("host.id")
	counter := meter.NewInt64Counter("a.counter", metric.WithKeys(key))
	labels := meter.Labels(key.String("value"))
	ctx := context.Background()
	counter.Add(ctx, 100, labels)

	v := counter.Bind(labels)
	v.Add(ctx, 20)

	m := counter.Measurement(1)
	meter.RecordBatch(ctx, labels, m)
	time.Sleep(2 * time.Minute)
}
