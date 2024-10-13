package metrics

import (
	"context"
	"fmt"
	"time"

	"github.com/riandyrn/otelchi"
	otelmetric "go.opentelemetry.io/otel/metric"
)

func NewRequestDurationMs() otelchi.MetricsRecorder {
	return &requestDurationMs{}
}

type requestDurationMs struct {
	requestDurationHistogram otelmetric.Int64Histogram
	startTime                time.Time
}

func (r *requestDurationMs) RegisterMetric(ctx context.Context, cfg otelchi.RegisterMetricConfig) {
	requestDurHistogram, err := cfg.Meter.Int64Histogram("request_duration_milliseconds")
	if err != nil {
		panic(fmt.Sprintf("failed to create request_duration_milliseconds histogram: %v", err))
	}
	r.requestDurationHistogram = requestDurHistogram
}

func (r *requestDurationMs) StartMetric(ctx context.Context, opts otelchi.MetricOpts) {
	r.startTime = time.Now()
}

func (r *requestDurationMs) EndMetric(ctx context.Context, opts otelchi.MetricOpts) {
	duration := time.Since(r.startTime)
	r.requestDurationHistogram.Record(ctx,
		int64(duration.Milliseconds()),
		opts.Measurement,
	)
}
