package utils

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.20.0"
	"go.opentelemetry.io/otel/trace"
)

func NewTracer(svcName string) (trace.Tracer, error) {
	// create otlp exporter, notice that here we are using insecure option
	// because we just want to export the trace locally, also notice that
	// here we don't set any endpoint because by default the otel will load
	// the endpoint from the environment variable `OTEL_EXPORTER_OTLP_ENDPOINT`
	exporter, err := otlptrace.New(
		context.Background(),
		otlptracehttp.NewClient(otlptracehttp.WithInsecure()),
	)
	if err != nil {
		return nil, fmt.Errorf("unable to initialize exporter due: %w", err)
	}
	// initialize tracer provider
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(svcName),
		)),
	)
	// set tracer provider and propagator properly, this is to ensure all
	// instrumentation library could run well
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))

	// returns tracer
	return otel.Tracer(svcName), nil
}
