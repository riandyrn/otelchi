package otelchi

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel/attribute"
	otelmetric "go.opentelemetry.io/otel/metric"
)

var (
	serviceKey = attribute.Key("service")
	idKey      = attribute.Key("id")
	methodKey  = attribute.Key("method")
	codeKey    = attribute.Key("code")
)

type httpReqProperties struct {
	Service string
	ID      string
	Method  string
	Code    int
}

func newMetricsRecorder(meter otelmetric.Meter) *metricsRecorder {
	httpRequestDurHistogram, err := meter.Int64Histogram("request_duration_seconds")
	if err != nil {
		panic(fmt.Sprintf("failed to create request_duration_seconds histogram: %v", err))
	}

	httpResponseSizeHistogram, err := meter.Int64Histogram("response_size_bytes")
	if err != nil {
		panic(fmt.Sprintf("failed to create response_size_bytes histogram: %v", err))
	}

	httpRequestsInflight, err := meter.Int64UpDownCounter("requests_inflight")
	if err != nil {
		panic(fmt.Sprintf("failed to create requests_inflight counter: %v", err))
	}

	return &metricsRecorder{
		httpRequestDurHistogram:   httpRequestDurHistogram,
		httpResponseSizeHistogram: httpResponseSizeHistogram,
		httpRequestsInflight:      httpRequestsInflight,
	}
}

type metricsRecorder struct {
	httpRequestDurHistogram   otelmetric.Int64Histogram
	httpResponseSizeHistogram otelmetric.Int64Histogram
	httpRequestsInflight      otelmetric.Int64UpDownCounter
}

func (r *metricsRecorder) RecordRequestDuration(ctx context.Context, p httpReqProperties, duration time.Duration) {
	r.httpRequestDurHistogram.Record(ctx,
		int64(duration.Seconds()),
		otelmetric.WithAttributes(
			serviceKey.String(p.Service),
			idKey.String(p.ID),
			methodKey.String(p.Method),
			codeKey.Int(p.Code),
		),
	)
}

func (r *metricsRecorder) RecordResponseSize(ctx context.Context, p httpReqProperties, size int64) {
	r.httpResponseSizeHistogram.Record(ctx,
		size,
		otelmetric.WithAttributes(
			serviceKey.String(p.Service),
			idKey.String(p.ID),
			methodKey.String(p.Method),
			codeKey.Int(p.Code),
		),
	)
}

func (r *metricsRecorder) RecordRequestsInflight(ctx context.Context, p httpReqProperties, count int64) {
	r.httpRequestsInflight.Add(ctx,
		count,
		otelmetric.WithAttributes(
			serviceKey.String(p.Service),
			idKey.String(p.ID),
			methodKey.String(p.Method),
		),
	)
}
