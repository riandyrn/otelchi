module github.com/riandyrn/otelchi/examples/basic

go 1.18

replace github.com/riandyrn/otelchi => ../../

require (
	github.com/go-chi/chi/v5 v5.0.12
	github.com/riandyrn/otelchi v0.6.0
	go.opentelemetry.io/otel v1.14.0
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.14.0
	go.opentelemetry.io/otel/sdk v1.14.0
	go.opentelemetry.io/otel/trace v1.14.0
)

require (
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/go-logr/logr v1.2.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	golang.org/x/sys v0.5.0 // indirect
)
