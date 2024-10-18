package metrics

import (
	"context"
	"fmt"

	otelmetric "go.opentelemetry.io/otel/metric"
)

const (
	metricNameRequestInFlight = "requests_inflight"
	metricUnitRequestInFlight = "{count}"
	metricDescRequestInFlight = "Measures the number of requests currently being processed by the server."
)

// [NewRequestInFlight] creates a new instance of [RequestInFlight].
func NewRequestInFlight() MetricsRecorder {
	return &RequestInFlight{}
}

// [RequestInFlight] is a metrics recorder for recording the number of requests in flight.
type RequestInFlight struct {
	requestInFlightCounter otelmetric.Int64UpDownCounter
}

// [RegisterMetric] registers the request in flight metrics recorder.
func (r *RequestInFlight) RegisterMetric(ctx context.Context, cfg RegisterMetricConfig) {
	requestInFlightCounter, err := cfg.Meter.Int64UpDownCounter(
		metricNameRequestInFlight,
		otelmetric.WithDescription(metricDescRequestInFlight),
		otelmetric.WithUnit(metricUnitRequestInFlight),
	)
	if err != nil {
		panic(fmt.Sprintf("unable to create %s counter: %v", metricNameRequestInFlight, err))
	}
	r.requestInFlightCounter = requestInFlightCounter
}

// [StartMetric] increments the number of requests in flight.
func (r *RequestInFlight) StartMetric(ctx context.Context, opts MetricOpts) {
	r.requestInFlightCounter.Add(ctx, 1, opts.Measurement)
}

// [EndMetric] decrements the number of requests in flight.
func (r *RequestInFlight) EndMetric(ctx context.Context, opts MetricOpts) {
	r.requestInFlightCounter.Add(ctx, -1, opts.Measurement)
}
