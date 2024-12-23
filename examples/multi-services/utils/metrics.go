package utils

import (
	"context"
	"go.opentelemetry.io/otel"
	"time"

	otelchimetric "github.com/riandyrn/otelchi/metric"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.20.0"
)

// NewMetricConfig creates metric configuration that includes:
// - Request Duration Metrics: measures the latency of HTTP requests
// - Request Inflight Metrics: tracks the number of concurrent requests
// - Response Size Metrics: measures the size of HTTP responses
func NewMetricConfig(serviceName string) (otelchimetric.BaseConfig, error) {
	// create context
	ctx := context.Background()

	// create otlp exporter using HTTP protocol. the endpoint will be loaded from
	// OTEL_EXPORTER_OTLP_METRICS_ENDPOINT environment variable
	exporter, err := otlpmetrichttp.New(
		ctx,
		otlpmetrichttp.WithInsecure(),
	)
	if err != nil {
		return otelchimetric.BaseConfig{}, err
	}

	// create resource with service name
	res, err := resource.New(
		ctx,
		resource.WithAttributes(
			semconv.ServiceName(serviceName),
		),
	)
	if err != nil {
		return otelchimetric.BaseConfig{}, err
	}

	// create meter provider with otlp exporter
	meterProvider := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(res),
		sdkmetric.WithReader(
			sdkmetric.NewPeriodicReader(
				exporter,
				sdkmetric.WithInterval(15*time.Second),
			),
		),
	)

	// set global meter provider
	otel.SetMeterProvider(meterProvider)

	// create and return base config for metrics with meter provider
	return otelchimetric.NewBaseConfig(serviceName,
		otelchimetric.WithMeterProvider(meterProvider),
	), nil
}
