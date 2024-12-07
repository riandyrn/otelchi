package utils

import (
	"net/http"

	otelchimetric "github.com/riandyrn/otelchi/metric"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/sdk/metric"
)

// NewMetricProvider creates a new metric middleware that includes:
// - Request Duration Metrics: measures the latency of HTTP requests
// - Request Inflight Metrics: tracks the number of concurrent requests
// - Response Size Metrics: measures the size of HTTP responses
func NewMetricProvider(serviceName string) (func(next http.Handler) http.Handler, error) {
	// Create Prometheus exporter
	exporter, err := prometheus.New()
	if err != nil {
		return nil, err
	}

	// Create meter provider with Prometheus exporter
	meterProvider := metric.NewMeterProvider(
		metric.WithReader(exporter),
	)

	// Set global meter provider
	otel.SetMeterProvider(meterProvider)

	// Create base config for metrics with meter provider
	cfg := otelchimetric.NewBaseConfig(serviceName,
		otelchimetric.WithMeterProvider(meterProvider),
	)

	// Create duration metrics middleware
	durationMiddleware := otelchimetric.NewRequestDurationMillis(cfg)

	// Create inflight requests metrics middleware
	inflightMiddleware := otelchimetric.NewRequestInFlight(cfg)

	// Create response size metrics middleware
	responseSizeMiddleware := otelchimetric.NewResponseSizeBytes(cfg)

	// Chain all middlewares together
	return func(next http.Handler) http.Handler {
		return durationMiddleware(
			inflightMiddleware(
				responseSizeMiddleware(next),
			),
		)
	}, nil
}
