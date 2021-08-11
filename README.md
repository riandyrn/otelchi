# otelchi

[![ci](https://github.com/riandyrn/otelchi/actions/workflows/ci.yaml/badge.svg?branch=main)](https://github.com/riandyrn/otelchi/actions/workflows/ci.yaml)
[![Go Report Card](https://goreportcard.com/badge/github.com/riandyrn/otelchi)](https://goreportcard.com/report/github.com/riandyrn/otelchi)
[![Documentation](https://godoc.org/github.com/riandyrn/otelchi?status.svg)](https://pkg.go.dev/mod/github.com/riandyrn/otelchi)

OpenTelemetry instrumentation for [go-chi/chi](https://github.com/go-chi/chi).

Essentialy it is adaptation from [otelmux](https://github.com/open-telemetry/opentelemetry-go-contrib/tree/main/instrumentation/github.com/gorilla/mux/otelmux) but instead using `gorilla/mux`, we use `go-chi/chi`.

Currently it could only instrument traces.
