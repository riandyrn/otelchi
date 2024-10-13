package metrics

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/felixge/httpsnoop"
	"go.opentelemetry.io/otel"
	otelmetric "go.opentelemetry.io/otel/metric"
	semconv "go.opentelemetry.io/otel/semconv/v1.20.0"
	"go.opentelemetry.io/otel/semconv/v1.20.0/httpconv"
)

const (
	metricNameResponseSizeBytes      = "response_size_bytes"
	metricUnitResponseSizeBytes      = "By"
	metricDescResponseSizeBytes      = "Measures the size of the response in bytes."
	metricSchemaURLResponseSizeBytes = semconv.SchemaURL
)

// [NewResponseSizeBytes] returns a middleware that measures the size of the response in bytes.
func NewResponseSizeBytes(serverName string, opts ...Option) func(next http.Handler) http.Handler {
	cfg := config{}
	for _, opt := range opts {
		opt.apply(&cfg)
	}

	if cfg.MeterProvider == nil {
		cfg.MeterProvider = otel.GetMeterProvider()
	}
	meter := cfg.MeterProvider.Meter(
		ScopeName,
		otelmetric.WithSchemaURL(metricSchemaURLResponseSizeBytes),
		otelmetric.WithInstrumentationVersion(Version()),
		otelmetric.WithInstrumentationAttributes(
			semconv.ServiceName(serverName),
		),
	)

	httpResponseSizeBytes, err := meter.Int64Histogram(
		metricNameResponseSizeBytes,
		otelmetric.WithDescription(metricDescResponseSizeBytes),
		otelmetric.WithUnit(metricUnitResponseSizeBytes),
	)
	if err != nil {
		panic(fmt.Sprintf("unable to create %s histogram due to: %v", metricNameResponseSizeBytes, err))
	}

	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			attributes := httpconv.ServerRequest(serverName, r)

			// get recording response writer
			rrw := getRRW(w)
			defer putRRW(rrw)

			next.ServeHTTP(w, r)

			// record the response size
			httpResponseSizeBytes.Record(ctx,
				int64(rrw.writtenBytes),
				otelmetric.WithAttributes(attributes...),
			)
		}
		return http.HandlerFunc(fn)
	}
}

// [recordingResponseWriter] is a wrapper around http.ResponseWriter that records the number of bytes written.
type recordingResponseWriter struct {
	writer       http.ResponseWriter
	written      bool
	writtenBytes int64
}

var rrwPool = &sync.Pool{
	New: func() interface{} {
		return &recordingResponseWriter{}
	},
}

func getRRW(writer http.ResponseWriter) *recordingResponseWriter {
	rrw := rrwPool.Get().(*recordingResponseWriter)
	rrw.written = false
	rrw.writtenBytes = 0
	rrw.writer = httpsnoop.Wrap(writer, httpsnoop.Hooks{
		Write: func(next httpsnoop.WriteFunc) httpsnoop.WriteFunc {
			return func(b []byte) (int, error) {
				if !rrw.written {
					rrw.written = true
					rrw.writtenBytes += int64(len(b))
				}
				return next(b)
			}
		},
		WriteHeader: func(next httpsnoop.WriteHeaderFunc) httpsnoop.WriteHeaderFunc {
			return func(statusCode int) {
				if !rrw.written {
					rrw.written = true
				}
				next(statusCode)
			}
		},
	})
	return rrw
}

func putRRW(rrw *recordingResponseWriter) {
	rrw.writer = nil
	rrwPool.Put(rrw)
}
