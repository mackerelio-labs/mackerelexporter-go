package mackerel

import (
	"go.opentelemetry.io/otel/metric/number"
	"go.opentelemetry.io/otel/sdk/export/metric/aggregation"
)

// https://github.com/open-telemetry/opentelemetry-go/pull/1412

// TODO: https://github.com/open-telemetry/opentelemetry-go/commit/49f699d65742e144cf19b5dd28f3d3a0891bf200#diff-520759597f6f3925842503aaacbaf225c8a8d2981a20c1c9732e13f26ca49325

func Quantile(p []aggregation.Point, q float64) (number.Number, error) {
	return number.Number{}, nil
}
