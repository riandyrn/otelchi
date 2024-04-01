module github.com/riandyrn/otelchi/examples/basic

go 1.15

replace github.com/riandyrn/otelchi => ../../

require (
	github.com/go-chi/chi/v5 v5.0.12
	github.com/riandyrn/otelchi v0.6.0
	go.opentelemetry.io/otel v1.10.0
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.10.0
	go.opentelemetry.io/otel/sdk v1.10.0
	go.opentelemetry.io/otel/trace v1.10.0
)
