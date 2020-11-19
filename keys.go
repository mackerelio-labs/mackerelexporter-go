package mackerel

import (
	"go.opentelemetry.io/otel/semconv"
)

// These keys are handled for creating hosts, graph-defs, or metrics.
var (
	// see https://github.com/open-telemetry/opentelemetry-specification/blob/master/specification/resource/semantic_conventions/README.md
	KeyServiceNS         = semconv.ServiceNamespaceKey
	KeyServiceName       = semconv.ServiceNameKey
	KeyServiceInstanceID = semconv.ServiceInstanceIDKey
	KeyServiceVersion    = semconv.ServiceVersionKey
	KeyHostID            = semconv.HostIDKey
	KeyHostName          = semconv.HostNameKey
	KeyCloudProvider     = semconv.CloudProviderKey
)
