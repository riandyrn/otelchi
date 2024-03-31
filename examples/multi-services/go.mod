module github.com/riandyrn/otelchi/examples/multi-services

go 1.15

replace github.com/riandyrn/otelchi => ../../

require (
	github.com/go-chi/chi/v5 v5.0.12
	github.com/riandyrn/otelchi v0.6.0
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.33.0
	go.opentelemetry.io/otel v1.10.0
	go.opentelemetry.io/otel/exporters/jaeger v1.9.0 // apparently this is the latest version that supports go1.15
	go.opentelemetry.io/otel/sdk v1.10.0
	go.opentelemetry.io/otel/trace v1.10.0
)

require golang.org/x/sys v0.0.0-20220520151302-bc2c85ada10a // indirect
