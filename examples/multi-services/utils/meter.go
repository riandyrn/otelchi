package utils

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.20.0"
)

func NewMeter(svcName string) error {
	// create otlp exporter, notice that here we are using insecure option
	// because we just want to export the trace locally, also notice that
	// here we don't set any endpoint because by default the otel will load
	// the endpoint from the environment variable `OTEL_EXPORTER_OTLP_ENDPOINT`
	exporter, err := otlpmetrichttp.New(
		context.Background(),
	)
	if err != nil {
		return fmt.Errorf("unable to initialize exporter due: %w", err)
	}
	// initialize tracer provider
	tp := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(exporter, sdkmetric.WithInterval(30*time.Second))),
		sdkmetric.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(svcName),
		)),
	)
	// set tracer provider and propagator properly, this is to ensure all
	// instrumentation library could run well
	otel.SetMeterProvider(tp)

	return nil
}
