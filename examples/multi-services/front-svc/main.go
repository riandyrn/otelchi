package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/riandyrn/otelchi"
	"github.com/riandyrn/otelchi/examples/multi-services/utils"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const (
	envKeyBackServiceURL    = "BACK_SERVICE_URL"
	envKeyJaegerEndpointURL = "JAEGER_ENDPOINT_URL"
	addr                    = ":8090"
	serviceName             = "front-svc"
)

func main() {
	// initialize tracer
	tracer, err := utils.NewTracer(serviceName, os.Getenv(envKeyJaegerEndpointURL))
	if err != nil {
		log.Fatalf("unable to initialize tracer due: %v", err)
	}
	// define router
	r := chi.NewRouter()
	r.Use(
		otelchi.Middleware(
			serviceName,
			otelchi.WithChiRoutes(r),
			otelchi.WithFilter(func(r *http.Request) bool {
				// ignore path "/"
				return r.URL.Path != "/"
			}),
		),
	)
	r.Get("/", utils.HealthCheckHandler)
	r.Get("/greet", func(w http.ResponseWriter, r *http.Request) {
		name, err := getName(r, tracer)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}
		w.Write([]byte(generateGreeting(r, tracer, name)))
	})
	// execute server
	log.Printf("front service is listening on %v", addr)
	err = http.ListenAndServe(addr, r)
	if err != nil {
		log.Fatalf("unable to execute server due: %v", err)
	}
}

func getName(r *http.Request, tracer trace.Tracer) (string, error) {
	// start span
	ctx, span := tracer.Start(r.Context(), "getName")
	defer span.End()

	// call back service, please note here we call the service using
	// instrumented http client
	ul := fmt.Sprintf(
		"%v/name?name=%v",
		os.Getenv(envKeyBackServiceURL),
		r.URL.Query().Get("name"),
	)
	resp, err := otelhttp.Get(ctx, ul)
	if err != nil {
		err = fmt.Errorf("unable to execute http request due: %w", err)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return "", err
	}
	defer resp.Body.Close()

	// read response body
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		err = fmt.Errorf("unable to read response data due: %w", err)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return "", err
	}

	return string(data), nil
}

func generateGreeting(r *http.Request, tracer trace.Tracer, name string) string {
	// start span
	_, span := tracer.Start(r.Context(), "generateGreeting")
	defer span.End()

	// generate greeting
	lang := r.URL.Query().Get("lang")
	switch lang {
	case "id":
		return fmt.Sprintf("Halo, %v! Kamu memilih bahasa: %v.", name, lang)
	default:
		return fmt.Sprintf("Hello, %v! You are choosing language: %v.", name, lang)
	}
}
