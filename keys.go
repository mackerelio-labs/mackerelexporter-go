package mackerel

import (
	"go.opentelemetry.io/otel/api/standard"
)

// These keys are handled for creating hosts, graph-defs, or metrics.
var (
	// see https://github.com/open-telemetry/opentelemetry-specification/blob/master/specification/resource/semantic_conventions/README.md
	KeyServiceNS         = standard.ServiceNamespaceKey
	KeyServiceName       = standard.ServiceNameKey
	KeyServiceInstanceID = standard.ServiceInstanceIDKey
	KeyServiceVersion    = standard.ServiceVersionKey
	KeyHostID            = standard.HostIDKey
	KeyHostName          = standard.HostNameKey
	KeyCloudProvider     = standard.CloudProviderKey
)
