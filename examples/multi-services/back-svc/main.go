package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/otel/trace"

	"github.com/riandyrn/otelchi"
	"github.com/riandyrn/otelchi/examples/multi-services/utils"
	otelchimetrics "github.com/riandyrn/otelchi/metrics"
)

const (
	addr        = ":8091"
	serviceName = "back-svc"
)

func main() {
	// init tracer provider
	tracer, err := utils.NewTracer(serviceName)
	if err != nil {
		log.Fatalf("unable to initialize tracer provider due: %v", err)
	}

	if err = utils.NewMeter(serviceName); err != nil {
		log.Fatalf("unable to initialize meter provider due: %v", err)
	}

	// define router
	r := chi.NewRouter()
	r.Use(otelchi.Middleware(
		serviceName,
		otelchi.WithChiRoutes(r),
		otelchi.WithMetricRecorders(otelchimetrics.NewRequestDurationMs()),
	))
	r.Get("/", utils.HealthCheckHandler)
	r.Get("/name", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(generateName(r.Context(), tracer)))
	})
	log.Printf("back service is listening on %v", addr)
	err = http.ListenAndServe(addr, r)
	if err != nil {
		log.Fatalf("unable to execute server due: %v", err)
	}
}

func generateName(ctx context.Context, tracer trace.Tracer) string {
	_, span := tracer.Start(ctx, "generateName")
	defer span.End()

	// simulate generate name with random delay
	time.Sleep(time.Duration(50+rand.Intn(50)) * time.Millisecond)

	rndNum := rand.New(rand.NewSource(time.Now().UnixNano())).Intn(100000)
	return fmt.Sprintf("user_%v", rndNum)
}
