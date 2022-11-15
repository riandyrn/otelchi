package main

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/riandyrn/otelchi"
	"github.com/riandyrn/otelchi/examples/multi-services/utils"
	"go.opentelemetry.io/otel/trace"
)

const (
	envKeyJaegerEndpointURL = "JAEGER_ENDPOINT_URL"
	addr                    = ":8091"
	serviceName             = "back-svc"
)

func main() {
	// init tracer provider
	tracer, err := utils.NewTracer(serviceName, os.Getenv(envKeyJaegerEndpointURL))
	if err != nil {
		log.Fatalf("unable to initialize tracer provider due: %v", err)
	}
	// define router
	r := chi.NewRouter()
	r.Use(otelchi.Middleware(serviceName, otelchi.WithChiRoutes(r)))
	r.Get("/", utils.HealthCheckHandler)
	r.Get("/name", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(generateName(r, tracer)))
	})
	log.Printf("back service is listening on %v", addr)
	err = http.ListenAndServe(addr, r)
	if err != nil {
		log.Fatalf("unable to execute server due: %v", err)
	}
}

func generateName(r *http.Request, tracer trace.Tracer) string {
	_, span := tracer.Start(r.Context(), "generateName")
	defer span.End()

	name := r.URL.Query().Get("name")
	if len(name) == 0 {
		rndNum := rand.New(rand.NewSource(time.Now().UnixNano())).Intn(100000)
		name = fmt.Sprintf("user_%v", rndNum)
	}

	return name
}
