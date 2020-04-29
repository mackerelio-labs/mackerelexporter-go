package mackerel

import (
	"go.opentelemetry.io/otel/api/core"
	"go.opentelemetry.io/otel/sdk/resource/resourcekeys"
)

// These keys are handled for creating hosts, graph-defs, or metrics.
var (
	// see https://github.com/open-telemetry/opentelemetry-specification/blob/master/specification/resource/semantic_conventions/README.md
	KeyServiceNS         = core.Key(resourcekeys.ServiceKeyNamespace)
	KeyServiceName       = core.Key(resourcekeys.ServiceKeyName)
	KeyServiceInstanceID = core.Key(resourcekeys.ServiceKeyInstanceID)
	KeyServiceVersion    = core.Key(resourcekeys.ServiceKeyVersion)
	KeyHostID            = core.Key(resourcekeys.HostKeyID)
	KeyHostName          = core.Key(resourcekeys.HostKeyName)
	KeyCloudProvider     = core.Key(resourcekeys.CloudKeyProvider)
)
