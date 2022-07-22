package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/riandyrn/otelchi"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"go.opentelemetry.io/otel/trace"
)

const (
	envKeyBackServiceURL    = "BACK_SERVICE_URL"
	envKeyJaegerEndpointURL = "JAEGER_ENDPOINT_URL"
	addr                    = ":8090"
)

func main() {
	// init tracer provider
	tp, err := initTracerProvider("front-svc", os.Getenv(envKeyJaegerEndpointURL))
	if err != nil {
		log.Fatalf("unable to initialize tracer provider due: %v", err)
	}
	// set global tracer provider & text propagators
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
	tracer := otel.Tracer("")
	// define router
	r := chi.NewRouter()
	r.Use(otelchi.Middleware("front-svc", otelchi.WithChiRoutes(r)))
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		name, err := getRandomName(r.Context(), tracer)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}
		w.Write([]byte(fmt.Sprintf("Hello, %s!", name)))
	})
	// execute server
	log.Printf("front service is listening on %v", addr)
	err = http.ListenAndServe(addr, r)
	if err != nil {
		log.Fatalf("unable to execute server due: %v", err)
	}
}

func getRandomName(ctx context.Context, tracer trace.Tracer) (string, error) {
	ctx, span := tracer.Start(ctx, "getRandomName")
	defer span.End()

	resp, err := otelhttp.Get(ctx, os.Getenv(envKeyBackServiceURL))
	if err != nil {
		err = fmt.Errorf("unable to execute http request due: %w", err)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return "", err
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		err = fmt.Errorf("unable to read response data due: %w", err)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return "", err
	}

	return string(data), nil
}

func initTracerProvider(svcName, jaegerEndpoint string) (*sdktrace.TracerProvider, error) {
	// create jaeger exporter
	exporter, err := jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint(jaegerEndpoint)))
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
	return tp, nil
}
