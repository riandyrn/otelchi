package metric

import (
	"fmt"
	"net/http"

	otelmetric "go.opentelemetry.io/otel/metric"
)

const (
	metricNameRequestInFlight = "requests_inflight"
	metricUnitRequestInFlight = "{count}"
	metricDescRequestInFlight = "Measures the number of requests currently being processed by the server."
)

// [RequestInFlight] is a metrics recorder for recording the number of requests in flight.
func NewRequestInFlight(cfg BaseConfig) func(next http.Handler) http.Handler {
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
			// define metric attributes
			attrs := otelmetric.WithAttributes(cfg.AttributesFunc(r)...)

			// increase the number of requests in flight
			counter.Add(r.Context(), 1, attrs)

			// execute next http handler
			next.ServeHTTP(w, r)

			// decrease the number of requests in flight
			counter.Add(r.Context(), -1, attrs)
		})
	}
}
