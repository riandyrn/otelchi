package metrics

import (
	"context"
	"fmt"
	"net/http"

	otelmetric "go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/semconv/v1.20.0/httpconv"
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

func NewResponseSizeBytesMiddleware(cfg MiddlewareConfig) func(next http.Handler) http.Handler {
	// init metric, here we are using histogram for capturing response size
	histogram, err := cfg.Meter.Int64Histogram(
		metricNameResponseSizeBytes,
		otelmetric.WithDescription(metricDescResponseSizeBytes),
		otelmetric.WithUnit(metricUnitResponseSizeBytes),
	)
	if err != nil {
		panic(fmt.Sprintf("unable to create %s histogram: %v", metricNameResponseSizeBytes, err))
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// get recording response writer
			rrw := getRRW(w)
			defer putRRW(rrw)

			// start metric before executing the handler
			next.ServeHTTP(rrw.writer, r)

			// end metric after executing the handler
			histogram.Record(
				r.Context(),
				int64(rrw.writtenBytes),
				otelmetric.WithAttributes(
					httpconv.ServerRequest(cfg.ServerName, r)...,
				),
			)
		})
	}
}
