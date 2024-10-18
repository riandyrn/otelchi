package metrics

import (
	"fmt"
	"net/http"
	"time"

	"go.opentelemetry.io/otel"
	otelmetric "go.opentelemetry.io/otel/metric"
	semconv "go.opentelemetry.io/otel/semconv/v1.20.0"
	"go.opentelemetry.io/otel/semconv/v1.20.0/httpconv"
)

const (
	metricNameRequestDurationMs      = "request_duration_milliseconds"
	metricUnitRequestDurationMs      = "ms"
	metricDescRequestDurationMs      = "Measures the latency of HTTP requests processed by the server, in milliseconds."
	metricSchemaURLRequestDurationMs = semconv.SchemaURL
)

// [NewRequestDurationMs] returns a middleware that measures the latency of HTTP requests processed by the server, in milliseconds.
func NewRequestDurationMs(serverName string, opts ...Option) func(next http.Handler) http.Handler {
	cfg := config{}
	for _, opt := range opts {
		opt.apply(&cfg)
	}

	if cfg.MeterProvider == nil {
		cfg.MeterProvider = otel.GetMeterProvider()
	}
	meter := cfg.MeterProvider.Meter(
		ScopeName,
		otelmetric.WithSchemaURL(metricSchemaURLRequestDurationMs),
		otelmetric.WithInstrumentationVersion(Version()),
		otelmetric.WithInstrumentationAttributes(
			semconv.ServiceName(serverName),
		),
	)

	httpRequestDurationMs, err := meter.Int64Histogram(
		metricNameRequestDurationMs,
		otelmetric.WithDescription(metricDescRequestDurationMs),
		otelmetric.WithUnit(metricUnitRequestDurationMs),
	)
	if err != nil {
		panic(fmt.Sprintf("unable to create %s histogram due to: %v", metricNameRequestDurationMs, err))
	}

	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			attributes := httpconv.ServerRequest(serverName, r)

			startTime := time.Now()

			next.ServeHTTP(w, r)

			// record the response size
			duration := time.Since(startTime)
			httpRequestDurationMs.Record(ctx,
				int64(duration.Milliseconds()),
				otelmetric.WithAttributes(attributes...),
			)
		}
		return http.HandlerFunc(fn)
	}
}
