package metrics

import (
	"fmt"
	"net/http"

	"go.opentelemetry.io/otel"
	otelmetric "go.opentelemetry.io/otel/metric"
	semconv "go.opentelemetry.io/otel/semconv/v1.20.0"
	"go.opentelemetry.io/otel/semconv/v1.20.0/httpconv"
)

const (
	metricNameRequestInFlight      = "requests_inflight"
	metricUnitRequestInFlight      = "{count}"
	metricDescRequestInFlight      = "Measures the number of requests currently being processed by the server."
	metricSchemaURLRequestInFlight = semconv.SchemaURL
)

// [NewRequestInFlight] returns a middleware that measures the number of requests currently being processed by the server.
func NewRequestInFlight(serverName string, opts ...Option) func(next http.Handler) http.Handler {
	cfg := config{}
	for _, opt := range opts {
		opt.apply(&cfg)
	}

	if cfg.MeterProvider == nil {
		cfg.MeterProvider = otel.GetMeterProvider()
	}
	meter := cfg.MeterProvider.Meter(
		ScopeName,
		otelmetric.WithSchemaURL(metricSchemaURLRequestInFlight),
		otelmetric.WithInstrumentationVersion(Version()),
		otelmetric.WithInstrumentationAttributes(
			semconv.ServiceName(serverName),
		),
	)

	requestsInFlightCounter, err := meter.Int64UpDownCounter(
		metricNameRequestInFlight,
		otelmetric.WithDescription(metricDescRequestInFlight),
		otelmetric.WithUnit(metricUnitRequestInFlight),
	)
	if err != nil {
		panic(fmt.Sprintf("unable to create %s counter due to: %v", metricNameRequestInFlight, err))
	}

	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			attributes := httpconv.ServerRequest(serverName, r)

			requestsInFlightCounter.Add(ctx, 1, otelmetric.WithAttributes(attributes...))

			next.ServeHTTP(w, r)

			requestsInFlightCounter.Add(ctx, -1, otelmetric.WithAttributes(attributes...))
		}
		return http.HandlerFunc(fn)
	}
}
