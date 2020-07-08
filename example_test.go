package mackerel_test

import (
	"log"
	"net/http"
	"os"

	"github.com/mackerelio-labs/mackerelexporter-go"
)

func ExampleInstallNewPipeline_pushMode() {
	apiKey := os.Getenv("MACKEREL_APIKEY")
	pusher, _, err := mackerel.InstallNewPipeline(
		mackerel.WithAPIKey(apiKey),
		mackerel.WithQuantiles([]float64{0.99, 0.90, 0.85}),
		mackerel.WithResource(
			mackerel.KeyHostID.String("1-2-3-4"),
			mackerel.KeyHostName.String("localhost"),
			mackerel.KeyServiceNS.String("service"),
			mackerel.KeyServiceName.String("role"),
		),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer pusher.Stop()
}

func ExampleInstallNewPipeline_pullMode() {
	pusher, handler, err := mackerel.InstallNewPipeline(
		mackerel.WithQuantiles([]float64{0.99, 0.90, 0.85}),
		mackerel.WithResource(
			mackerel.KeyHostID.String("1-2-3-4"),
			mackerel.KeyHostName.String("localhost"),
			mackerel.KeyServiceNS.String("service"),
			mackerel.KeyServiceName.String("role"),
		),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer pusher.Stop()

	http.HandleFunc("/metrics", handler)
}
