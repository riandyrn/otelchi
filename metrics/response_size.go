package metrics

import (
	"context"
	"fmt"

	"github.com/riandyrn/otelchi"
	otelmetric "go.opentelemetry.io/otel/metric"
)

// [NewResponseSizeBytes] creates a new instance of [responseSizeBytes].
func NewResponseSizeBytes() otelchi.MetricsRecorder {
	return &responseSizeBytes{}
}

// [responseSizeBytes] is a metrics recorder for recording response size in bytes.
type responseSizeBytes struct {
	responseSizeBytes otelmetric.Int64Histogram
}

func (r *responseSizeBytes) RegisterMetric(ctx context.Context, cfg otelchi.RegisterMetricConfig) {
	httpResponseSizeBytes, err := cfg.Meter.Int64Histogram("response_size_bytes")
	if err != nil {
		panic(fmt.Sprintf("unable to create response_size_bytes histogram: %v", err))
	}
	r.responseSizeBytes = httpResponseSizeBytes
}

func (r *responseSizeBytes) StartMetric(ctx context.Context, opts otelchi.MetricOpts) {}

func (r *responseSizeBytes) EndMetric(ctx context.Context, opts otelchi.MetricOpts) {
	r.responseSizeBytes.Record(ctx,
		int64(opts.ResponseData.WrittenBytes),
		opts.Measurement,
	)
}
