module github.com/riandyrn/otelchi/examples/multi-services

go 1.18

require (
	github.com/go-chi/chi/v5 v5.0.7
	github.com/riandyrn/otelchi v0.5.0
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.36.4
	go.opentelemetry.io/otel v1.11.1
	go.opentelemetry.io/otel/exporters/jaeger v1.11.1
	go.opentelemetry.io/otel/sdk v1.11.1
	go.opentelemetry.io/otel/trace v1.11.1
)

require (
	github.com/felixge/httpsnoop v1.0.3 // indirect
	github.com/go-logr/logr v1.2.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	go.opentelemetry.io/contrib v1.11.1 // indirect
	go.opentelemetry.io/otel/metric v0.33.0 // indirect
	golang.org/x/sys v0.2.0 // indirect
)
