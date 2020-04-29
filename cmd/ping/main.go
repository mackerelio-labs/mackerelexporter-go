package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"runtime"
	"time"

	"go.opentelemetry.io/otel/api/core"
	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/api/metric"
	"go.opentelemetry.io/otel/api/unit"

	"github.com/lufia/mackerelexporter-go"
)

// https://github.com/open-telemetry/opentelemetry-go/blob/master/sdk/metric/example_test.go

var (
	hints = []string{
		"http.handlers.#.latency",
	}

	quantiles = []float64{0.99, 0.90, 0.85}

	meter     = global.MeterProvider().Meter("example/ping")
	meterMust = metric.Must(meter)
	memAlloc  = meterMust.RegisterInt64Observer("runtime.memory.alloc", func(result metric.Int64ObserverResult) {
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		result.Observe(int64(m.Alloc), hostLabels...)
	}, metric.WithUnit(unit.Bytes))
	latency = meterMust.NewFloat64Measure("http.handlers.index.latency")

	hostLabels = []core.KeyValue{
		mackerel.KeyHostID.String("10-1-2-241"),
		mackerel.KeyHostName.String("localhost"),
	}

	requestCount = meterMust.NewInt64Counter("http.requests.count")

	serviceLabels = []core.KeyValue{
		mackerel.KeyServiceNS.String("example"),
		mackerel.KeyServiceName.String("ping"),
	}
)

func indexHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	t0 := time.Now()
	fmt.Fprintf(w, "OK\n")

	latency.Record(ctx, time.Since(t0).Seconds(), hostLabels...)
	requestCount.Add(ctx, 1)
}

var (
	flagDebug = flag.Bool("debug", false, "enables logs for debugging")
)

func main() {
	log.SetFlags(0)
	flag.Parse()
	apiKey := os.Getenv("MACKEREL_APIKEY")
	opts := []mackerel.Option{
		mackerel.WithAPIKey(apiKey),
		mackerel.WithQuantiles(quantiles),
		mackerel.WithHints(hints),
		mackerel.WithResource(serviceLabels...),
	}
	if *flagDebug {
		opts = append(opts, mackerel.WithDebug())
	}
	pusher, handler, err := mackerel.InstallNewPipeline(opts...)
	if err != nil {
		log.Fatal(err)
	}
	defer pusher.Stop()

	http.HandleFunc("/", indexHandler)
	if handler != nil {
		http.HandleFunc("/metrics", handler)
	}
	http.ListenAndServe(":8080", nil)
}
