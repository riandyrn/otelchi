package metrics

import (
	"context"
	"fmt"
	"net/http"
	"time"

	otelmetric "go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/semconv/v1.20.0/httpconv"
)

const (
	metricNameRequestDurationMs = "request_duration_milliseconds"
	metricUnitRequestDurationMs = "ms"
	metricDescRequestDurationMs = "Measures the latency of HTTP requests processed by the server, in milliseconds."
)

// [NewRequestDurationMs] creates a new instance of [RequestDurationMs].
func NewRequestDurationMs() MetricsRecorder {
	return &RequestDurationMs{}
}

// [RequestDurationMs] is a metrics recorder for recording request duration in milliseconds.
type RequestDurationMs struct {
	requestDurationHistogram otelmetric.Int64Histogram
	startTime                time.Time
}

// [RegisterMetric] registers the request duration metrics recorder.
func (r *RequestDurationMs) RegisterMetric(ctx context.Context, cfg RegisterMetricConfig) {
	requestDurationHistogram, err := cfg.Meter.Int64Histogram(
		metricNameRequestDurationMs,
		otelmetric.WithDescription(metricDescRequestDurationMs),
		otelmetric.WithUnit(metricUnitRequestDurationMs),
	)
	if err != nil {
		panic(fmt.Sprintf("unable to create %s histogram: %v", metricNameRequestDurationMs, err))
	}
	r.requestDurationHistogram = requestDurationHistogram
}

// [StartMetric] starts the request duration metrics recorder.
func (r *RequestDurationMs) StartMetric(ctx context.Context, opts MetricOpts) {
	r.startTime = time.Now()
}

// [EndMetric] ends the request duration metrics recorder.
func (r *RequestDurationMs) EndMetric(ctx context.Context, opts MetricOpts) {
	duration := time.Since(r.startTime)
	r.requestDurationHistogram.Record(ctx,
		int64(duration.Milliseconds()),
		opts.Measurement,
	)
}

func NewRequestDurationMillisMiddleware(cfg MiddlewareConfig) func(next http.Handler) http.Handler {
	// init metric, here we are using histogram for capturing request duration
	histogram, err := cfg.Meter.Int64Histogram(
		metricNameRequestDurationMs,
		otelmetric.WithDescription(metricDescRequestDurationMs),
		otelmetric.WithUnit(metricUnitRequestDurationMs),
	)
	if err != nil {
		panic(fmt.Sprintf("unable to create %s histogram: %v", metricNameRequestDurationMs, err))
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// get recording response writer
			rrw := getRRW(w)
			defer putRRW(rrw)

			// start metric before executing the handler
			startTime := time.Now()

			// execute next http handler
			next.ServeHTTP(rrw.writer, r)

			// end metric after executing the handler
			duration := time.Since(startTime)
			histogram.Record(
				r.Context(),
				int64(duration.Milliseconds()),
				otelmetric.WithAttributes(
					httpconv.ServerRequest(cfg.ServerName, r)...,
				),
			)
		})
	}
}
