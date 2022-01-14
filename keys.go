package mackerel

import (
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
)

// These keys are handled for creating hosts, graph-defs, or metrics.
var (
	// see https://github.com/open-telemetry/opentelemetry-specification/blob/main/specification/resource/semantic_conventions/README.md
	KeyServiceNS         = semconv.ServiceNamespaceKey
	KeyServiceName       = semconv.ServiceNameKey
	KeyServiceInstanceID = semconv.ServiceInstanceIDKey
	KeyServiceVersion    = semconv.ServiceVersionKey
	KeyHostID            = semconv.HostIDKey
	KeyHostName          = semconv.HostNameKey
	KeyCloudProvider     = semconv.CloudProviderKey
)
