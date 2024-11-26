package metric

import (
	"net/http"
	"sync"

	"github.com/felixge/httpsnoop"
	"go.opentelemetry.io/otel"
	otelmetric "go.opentelemetry.io/otel/metric"
	semconv "go.opentelemetry.io/otel/semconv/v1.20.0"
)

const (
	ScopeName = "github.com/riandyrn/otelchi/metric"
)

// BaseConfig is used to configure the metrics middleware.
type BaseConfig struct {
	// for initialization
	meterProvider otelmetric.MeterProvider

	// actual config state
	Meter      otelmetric.Meter
	ServerName string
}

// Option specifies instrumentation configuration options.
type Option interface {
	apply(*BaseConfig)
}

type optionFunc func(*BaseConfig)

func (o optionFunc) apply(c *BaseConfig) {
	o(c)
}

// WithMeterProvider specifies a meter provider to use for creating a meter.
// If none is specified, the global provider is used.
func WithMeterProvider(provider otelmetric.MeterProvider) Option {
	return optionFunc(func(cfg *BaseConfig) {
		cfg.meterProvider = provider
	})
}

func NewBaseConfig(serverName string, opts ...Option) BaseConfig {
	// init base config
	cfg := BaseConfig{
		ServerName: serverName,
	}
	for _, opt := range opts {
		opt.apply(&cfg)
	}

	if cfg.meterProvider == nil {
		cfg.meterProvider = otel.GetMeterProvider()
	}
	cfg.Meter = cfg.meterProvider.Meter(
		ScopeName,
		otelmetric.WithSchemaURL(semconv.SchemaURL),
		otelmetric.WithInstrumentationVersion(Version()),
		otelmetric.WithInstrumentationAttributes(
			semconv.ServiceName(serverName),
		),
	)

	return cfg
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
