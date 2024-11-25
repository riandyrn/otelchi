package metrics

import (
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

func NewResponseSizeBytesMiddleware(cfg Config) func(next http.Handler) http.Handler {
	// init metric, here we are using histogram for capturing response size
	histogram, err := cfg.meter.Int64Histogram(
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
					httpconv.ServerRequest(cfg.serverName, r)...,
				),
			)
		})
	}
}
