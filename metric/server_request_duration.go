package metric

import (
	"fmt"
	"net/http"

	"github.com/felixge/httpsnoop"
	otelmetric "go.opentelemetry.io/otel/metric"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

const (
	metricNameServerRequestDuration = "http.server.request.duration"
	metricUnitServerRequestDuration = "s"
	metricDescServerRequestDuration = "Duration of HTTP server requests."
)

// NewServerRequestDuration records the duration of HTTP server requests in seconds.
func NewServerRequestDuration(cfg BaseConfig) func(next http.Handler) http.Handler {
	histogram, err := cfg.Meter.Float64Histogram(
		metricNameServerRequestDuration,
		otelmetric.WithDescription(metricDescServerRequestDuration),
		otelmetric.WithUnit(metricUnitServerRequestDuration),

		// Use the same explicit bucket boundaries as otelhttp so request duration
		// histograms are useful for common HTTP latency ranges and stay consistent
		// with other OpenTelemetry HTTP server instrumentation.
		otelmetric.WithExplicitBucketBoundaries(
			0.005, 0.01, 0.025, 0.05, 0.075, 0.1,
			0.25, 0.5, 0.75, 1, 2.5, 5, 7.5, 10,
		),
	)
	if err != nil {
		panic(fmt.Sprintf("unable to create %s histogram: %v", metricNameServerRequestDuration, err))
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// CaptureMetrics runs next and reports the final status code (defaulting
			// to 200 when the handler never calls WriteHeader) and the duration.
			metrics := httpsnoop.CaptureMetrics(next, w, r)

			attrs := cfg.AttributesFunc(r)
			attrs = append(attrs, semconv.HTTPResponseStatusCode(metrics.Code))

			histogram.Record(
				r.Context(),
				metrics.Duration.Seconds(),
				otelmetric.WithAttributes(attrs...),
			)
		})
	}
}
