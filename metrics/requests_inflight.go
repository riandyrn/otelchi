package metrics

import (
	"context"
	"fmt"

	"github.com/riandyrn/otelchi"
	otelmetric "go.opentelemetry.io/otel/metric"
)

type ResponseInFlightConfig struct {
	Meter otelmetric.Meter
}

func NewResponseInFlight() otelchi.MetricsRecorder {
	return &requestInFlight{}
}

type requestInFlight struct {
	requestsInFlightCounter otelmetric.Int64UpDownCounter
}

func (r *requestInFlight) RegisterMetric(ctx context.Context, cfg otelchi.RegisterMetricConfig) {
	requestsInFlightCounter, err := cfg.Meter.Int64UpDownCounter("requests_inflight")
	if err != nil {
		panic(fmt.Sprintf("failed to create requests_inflight histogram: %v", err))
	}
	r.requestsInFlightCounter = requestsInFlightCounter
}

func (r *requestInFlight) StartMetric(ctx context.Context, opts otelchi.MetricOpts) {
	r.requestCount(ctx, opts.Measurement, 1)
}

func (r *requestInFlight) EndMetric(ctx context.Context, opts otelchi.MetricOpts) {
	r.requestCount(ctx, opts.Measurement, -1)
}

func (r *requestInFlight) requestCount(ctx context.Context, attributes otelmetric.AddOption, count int64) {
	r.requestsInFlightCounter.Add(ctx, count, attributes)
}
