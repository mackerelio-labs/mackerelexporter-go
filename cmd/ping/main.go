package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"runtime"
	"time"

	"go.opentelemetry.io/otel/api/core"
	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/api/key"
	"go.opentelemetry.io/otel/api/metric"

	"github.com/lufia/mackerelexporter-go"
)

// https://github.com/open-telemetry/opentelemetry-go/blob/master/sdk/metric/example_test.go

var (
	keyHostID      = key.New("host.id")           // custom identifier
	keyHostName    = key.New("host.name")         // hostname
	keyServiceNS   = key.New("service.namespace") // service
	keyServiceName = key.New("service.name")      // role

	keys = []core.Key{
		keyHostID,
		keyHostName,
		keyServiceNS,
		keyServiceName,
	}

	hints = []string{
		"http.handlers.#.latency",
	}

	quantiles = []float64{0.99, 0.90, 0.85}

	meter     = global.MeterProvider().Meter("example/ping")
	meterMust = metric.Must(meter)
	memAlloc  = meterMust.RegisterInt64Observer("runtime.memory.alloc", func(result metric.Int64ObserverResult) {
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		result.Observe(int64(m.Alloc), labels...)
	}, metric.WithKeys(keys...))
	latency = meterMust.NewFloat64Measure("http.handlers.index.latency", metric.WithKeys(keys...))

	labels = []core.KeyValue{
		keyHostID.String("10-1-2-241"),
		keyHostName.String("localhost"),
		keyServiceNS.String("example"),
		keyServiceName.String("ping"),
	}

	requestCount = meterMust.NewInt64Counter("http.requests.count", metric.WithKeys(keys...))

	serviceLabels = []core.KeyValue{
		keyServiceNS.String("example"),
		keyServiceName.String("ping"),
	}
)

func indexHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	t0 := time.Now()
	fmt.Fprintf(w, "OK\n")

	latency.Record(ctx, time.Since(t0).Seconds(), labels...)
	requestCount.Add(ctx, 1, serviceLabels...)
}

func main() {
	log.SetFlags(0)
	apiKey := os.Getenv("MACKEREL_APIKEY")
	opts := []mackerel.Option{
		mackerel.WithAPIKey(apiKey),
		mackerel.WithQuantiles(quantiles),
		mackerel.WithHints(hints),
	}
	pusher, err := mackerel.InstallNewPipeline(opts...)
	if err != nil {
		log.Fatal(err)
	}
	defer pusher.Stop()

	http.HandleFunc("/", indexHandler)
	http.ListenAndServe(":8080", nil)
}
