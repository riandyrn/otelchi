package metrics

import (
	"context"
	"fmt"
	"time"

	otelmetric "go.opentelemetry.io/otel/metric"
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
