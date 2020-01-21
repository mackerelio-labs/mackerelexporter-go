package main

import (
	"context"
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

	"github.com/lufia/mackerelexporter"
)

// https://github.com/open-telemetry/opentelemetry-go/blob/master/sdk/metric/example_test.go

var (
	keyHostID   = key.New("host.id")   // custom identifier
	keyHostName = key.New("host.name") // hostname

	keys = []core.Key{
		keyHostID,
		keyHostName,
	}

	hints = []string{
		"http.handlers.#.latency",
	}

	quantiles = []float64{0.99, 0.90, 0.85}

	meter    = global.MeterProvider().Meter("example/ping")
	gcCount  = meter.NewInt64Counter("runtime.gc.count", metric.WithKeys(keys...))
	memAlloc = meter.NewInt64Gauge("runtime.memory.alloc", metric.WithKeys(keys...))
	latency  = meter.NewFloat64Measure("http.handlers.index.latency", metric.WithKeys(keys...))

	labels = meter.Labels(
		keyHostID.String("10-1-2-241"),
		keyHostName.String("localhost"),
	)
)

func startStats(ctx context.Context) {
	var last uint32
	go func() {
		tick := time.NewTicker(10 * time.Second)
		defer tick.Stop()
		for {
			select {
			case <-tick.C:
				var m runtime.MemStats
				runtime.ReadMemStats(&m)
				alloc := memAlloc.Measurement(int64(m.Alloc))
				gc := gcCount.Measurement(int64(m.NumGC - last))
				meter.RecordBatch(ctx, labels, alloc, gc)
				last = m.NumGC
			case <-ctx.Done():
				return
			}
		}
	}()
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	t0 := time.Now()
	fmt.Fprintf(w, "OK")

	latency.Record(ctx, time.Since(t0).Seconds(), meter.Labels(
		keyHostID.String("10-1-2-241"),
		keyHostName.String("localhost"),
	))
}

func main() {
	log.SetFlags(0)
	apiKey := os.Getenv("MACKEREL_APIKEY")
	pusher, err := mackerel.InstallNewPipeline(
		mackerel.WithAPIKey(apiKey),
		mackerel.WithQuantiles(quantiles),
		mackerel.WithHints(hints),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer pusher.Stop()

	startStats(context.Background())
	http.HandleFunc("/", indexHandler)
	http.ListenAndServe(":8080", nil)
}
