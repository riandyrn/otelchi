package utils

import (
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.20.0"
)

func NewMeter(svcName string) error {
	// create otlp exporter
	// notice that here we are using stdout exporter for simplicity
	exporter, err := stdoutmetric.New(stdoutmetric.WithPrettyPrint())
	if err != nil {
		return fmt.Errorf("unable to create stdout exporter: %w", err)
	}

	// initialize tracer provider
	reader := sdkmetric.NewPeriodicReader(exporter, sdkmetric.WithInterval(30*time.Second))
	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(reader),
		sdkmetric.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(svcName),
		)),
	)
	// set tracer provider and propagator properly, this is to ensure all
	// instrumentation library could run well
	otel.SetMeterProvider(mp)

	return nil
}
