package metrics

import (
	"context"
	"fmt"
	"net/http"

	otelmetric "go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/semconv/v1.20.0/httpconv"
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

func NewRequestInFlightMiddleware(cfg MiddlewareConfig) func(next http.Handler) http.Handler {
	// init metric, here we are using counter for capturing request in flight
	counter, err := cfg.Meter.Int64UpDownCounter(
		metricNameRequestInFlight,
		otelmetric.WithDescription(metricDescRequestInFlight),
		otelmetric.WithUnit(metricUnitRequestInFlight),
	)
	if err != nil {
		panic(fmt.Sprintf("unable to create %s counter: %v", metricNameRequestInFlight, err))
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// get recording response writer
			rrw := getRRW(w)
			defer putRRW(rrw)

			// start metric before executing the handler
			counter.Add(r.Context(), 1, otelmetric.WithAttributes(
				httpconv.ServerRequest(cfg.ServerName, r)...,
			))

			// execute next http handler
			next.ServeHTTP(rrw.writer, r)

			// end metric after executing the handler
			counter.Add(r.Context(), -1, otelmetric.WithAttributes(
				httpconv.ServerRequest(cfg.ServerName, r)...,
			))
		})
	}
}
