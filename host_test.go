package mackerel

import (
	"testing"

	"go.opentelemetry.io/otel/api/core"
)

func TestUnmarshalHost(t *testing.T) {
	labels := []core.KeyValue{
		core.Key("service.name").String("test"),
	}
}
