package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"go.opentelemetry.io/otel/api/core"
	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/api/key"
	"go.opentelemetry.io/otel/api/metric"

	"github.com/lufia/mackerelexporter"
)

// https://github.com/open-telemetry/opentelemetry-go/blob/master/sdk/metric/example_test.go

var (
	keyHostID      = key.New("host.id")               // custom identifier
	keyHostName    = key.New("host.name")             // hostname
	keyGraphClass  = key.New("mackerel.graph.class")  // graph-def's name
	keyMetricClass = key.New("mackerel.metric.class") // graph-def's metric name

	keys = []core.Key{
		keyHostID,
		keyHostName,
		keyGraphClass,
		keyMetricClass,
	}

	meter   = global.MeterProvider().Meter("example/ping")
	counter = meter.NewInt64Counter("handlers.requests.count", metric.WithKeys(keys...))
	measure = meter.NewFloat64Measure("handlers.index.latency", metric.WithKeys(keys...))
	gauge   = meter.NewInt64Gauge("handlers.last_accessed", metric.WithKeys(keys...))
)

func indexHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	t0 := time.Now()
	fmt.Fprintf(w, "OK")

	counter.Add(ctx, 1, meter.Labels(
		keyHostID.String("10-1-2-241"),
		keyHostName.String("localhost"),
		keyGraphClass.String("handlers.requests.*"),
		keyMetricClass.String("handlers.requests.*"),
	))
	measure.Record(ctx, time.Since(t0).Seconds(), meter.Labels(
		keyHostID.String("10-1-2-241"),
		keyHostName.String("localhost"),
		keyGraphClass.String("handlers.#"),
		keyMetricClass.String("handlers.#.latency"),
	))
	gauge.Set(ctx, time.Now().Unix(), meter.Labels(
		keyHostID.String("10-1-2-241"),
		keyHostName.String("localhost"),
		keyGraphClass.String("handlers.last_accessed"),
		keyMetricClass.String("handlers.last_accessed"),
	))
}

func main() {
	log.SetFlags(0)
	apiKey := os.Getenv("MACKEREL_APIKEY")
	pusher, err := mackerel.InstallNewPipeline(
		mackerel.WithAPIKey(apiKey),
		mackerel.WithQuantiles([]float64{0.99, 0.90, 0.85}),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer pusher.Stop()

	http.HandleFunc("/", indexHandler)
	http.ListenAndServe(":8080", nil)
}
