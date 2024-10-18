package metrics

import (
	"context"
	"net/http"
	"sync"

	"github.com/felixge/httpsnoop"
	"go.opentelemetry.io/otel"

	otelmetric "go.opentelemetry.io/otel/metric"
	semconv "go.opentelemetry.io/otel/semconv/v1.20.0"
	"go.opentelemetry.io/otel/semconv/v1.20.0/httpconv"
)

// [Middleware] sets up a handler to record metrics for the incoming requests.
func Middleware(serverName string, opts ...Option) func(next http.Handler) http.Handler {
	cfg := config{}
	for _, opt := range opts {
		opt.apply(&cfg)
	}

	if cfg.MeterProvider == nil {
		cfg.MeterProvider = otel.GetMeterProvider()
	}
	meter := cfg.MeterProvider.Meter(
		ScopeName,
		otelmetric.WithSchemaURL(semconv.SchemaURL),
		otelmetric.WithInstrumentationVersion(Version()),
		otelmetric.WithInstrumentationAttributes(
			semconv.ServiceName(serverName),
		),
	)

	// the reason why we register metrics recorders here
	// is because if panic triggered, it happens during the initialization
	// of the middleware, not during the request processing.
	for _, recorder := range cfg.MetricRecorders {
		recorder.RegisterMetric(context.Background(), RegisterMetricConfig{
			Meter: meter,
		})
	}

	return func(handler http.Handler) http.Handler {
		return &metricware{
			serverName:      serverName,
			meter:           meter,
			metricRecorders: cfg.MetricRecorders,
			handler:         handler,
		}
	}
}

type metricware struct {
	serverName      string
	meter           otelmetric.Meter
	metricRecorders []MetricsRecorder
	handler         http.Handler
}

// [ServeHTTP] implements the [http.Handler] interface.
// It does the actual metric recording.
func (ow *metricware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	attributes := httpconv.ServerRequest(ow.serverName, r)

	// get recording response writer
	rrw := getRRW(w)
	defer putRRW(rrw)

	// start metric before executing the handler
	metricOpts := MetricOpts{
		Measurement: otelmetric.WithAttributes(attributes...),
	}
	for _, recorder := range ow.metricRecorders {
		recorder.StartMetric(ctx, metricOpts)
	}

	// execute next http handler
	ow.handler.ServeHTTP(rrw.writer, r)

	// end metric after executing the handler
	metricOpts.ResponseData = ResponseData{
		WrittenBytes: rrw.writtenBytes,
	}
	for _, recorder := range ow.metricRecorders {
		recorder.EndMetric(ctx, metricOpts)
	}
}

// [recordingResponseWriter] is a wrapper around [http.ResponseWriter] that records the number of bytes written.
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
