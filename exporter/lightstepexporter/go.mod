module github.com/open-telemetry/opentelemetry-collector-contrib/exporter/lightstepexporter

go 1.12

require (
	github.com/census-instrumentation/opencensus-proto v0.2.1
	github.com/lightstep/lightstep-tracer-go v0.19.0
	github.com/lightstep/opentelemetry-exporter-go v0.1.3
	github.com/open-telemetry/opentelemetry-collector v0.2.6
	github.com/stretchr/testify v1.5.1
	go.opentelemetry.io/otel v0.2.3
	go.uber.org/zap v1.14.0
)

replace git.apache.org/thrift.git v0.12.0 => github.com/apache/thrift v0.12.0