package metrics

import (
	"context"
	"fmt"

	otelmetric "go.opentelemetry.io/otel/metric"
)

const (
	metricNameResponseSizeBytes = "response_size_bytes"
	metricUnitResponseSizeBytes = "By"
	metricDescResponseSizeBytes = "Measures the size of the response in bytes."
)

// [NewResponseSizeBytes] creates a new instance of [ResponseSizeBytes].
func NewResponseSizeBytes() MetricsRecorder {
	return &ResponseSizeBytes{}
}

// [ResponseSizeBytes] is a metrics recorder for recording response size in bytes.
type ResponseSizeBytes struct {
	responseSizeBytesHistogram otelmetric.Int64Histogram
}

// [RegisterMetric] registers the response size metrics recorder.
func (r *ResponseSizeBytes) RegisterMetric(ctx context.Context, cfg RegisterMetricConfig) {
	responseSizeBytesHistogram, err := cfg.Meter.Int64Histogram(
		metricNameResponseSizeBytes,
		otelmetric.WithDescription(metricDescResponseSizeBytes),
		otelmetric.WithUnit(metricUnitResponseSizeBytes),
	)
	if err != nil {
		panic(fmt.Sprintf("unable to create %s histogram: %v", metricNameResponseSizeBytes, err))
	}
	r.responseSizeBytesHistogram = responseSizeBytesHistogram
}

// [StartMetric] starts the response size metrics recorder, currently does nothing.
func (r *ResponseSizeBytes) StartMetric(ctx context.Context, opts MetricOpts) {}

// [EndMetric] records the written response size in bytes.
func (r *ResponseSizeBytes) EndMetric(ctx context.Context, opts MetricOpts) {
	r.responseSizeBytesHistogram.Record(ctx,
		int64(opts.ResponseData.WrittenBytes),
		opts.Measurement,
	)
}
