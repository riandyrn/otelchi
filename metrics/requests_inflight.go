package metrics

import (
	"context"
	"fmt"

	"github.com/riandyrn/otelchi"
	otelmetric "go.opentelemetry.io/otel/metric"
)

// [NewResponseInFlight] creates a new instance of [requestInFlight].
func NewResponseInFlight() otelchi.MetricsRecorder {
	return &requestInFlight{}
}

// [requestInFlight] is a metrics recorder for recording requests in flight.
// It records the number of requests currently being processed by the server.
type requestInFlight struct {
	requestsInFlightCounter otelmetric.Int64UpDownCounter
}

func (r *requestInFlight) RegisterMetric(ctx context.Context, cfg otelchi.RegisterMetricConfig) {
	requestsInFlightCounter, err := cfg.Meter.Int64UpDownCounter("requests_inflight")
	if err != nil {
		panic(fmt.Sprintf("unable to create requests_inflight histogram: %v", err))
	}
	r.requestsInFlightCounter = requestsInFlightCounter
}

func (r *requestInFlight) StartMetric(ctx context.Context, opts otelchi.MetricOpts) {
	r.requestsInFlightCounter.Add(ctx, 1, opts.Measurement)
}

func (r *requestInFlight) EndMetric(ctx context.Context, opts otelchi.MetricOpts) {
	r.requestsInFlightCounter.Add(ctx, -1, opts.Measurement)
}
