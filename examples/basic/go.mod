module github.com/riandyrn/otelchi/examples/basic

go 1.22.0

replace github.com/riandyrn/otelchi => ../../

require (
	github.com/go-chi/chi/v5 v5.1.0
	github.com/riandyrn/otelchi v0.10.0
	go.opentelemetry.io/otel v1.28.0
	go.opentelemetry.io/otel v1.30.0
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.30.0
	go.opentelemetry.io/otel/sdk v1.30.0
	go.opentelemetry.io/otel/trace v1.30.0
)

require (
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/go-logr/logr v1.4.2 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/google/uuid v1.6.0 // indirect
	go.opentelemetry.io/otel/metric v1.30.0 // indirect
	golang.org/x/sys v0.25.0 // indirect
)
