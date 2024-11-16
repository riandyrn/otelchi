package metrics

import (
	"context"

	otelmetric "go.opentelemetry.io/otel/metric"
)

const (
	ScopeName = "github.com/riandyrn/otelchi/metrics"
)

// config is used to configure the metrics middleware.
type config struct {
	MeterProvider   otelmetric.MeterProvider
	MetricRecorders []MetricsRecorder
}

// Option specifies instrumentation configuration options.
type Option interface {
	apply(*config)
}

type optionFunc func(*config)

func (o optionFunc) apply(c *config) {
	o(c)
}

// [RegisterMetricConfig] is used to configure metric registration.
type RegisterMetricConfig struct {
	Meter otelmetric.Meter
}

// [ResponseData] is used to store response metrics data.
type ResponseData struct {
	WrittenBytes int64
}

// [MetricOpts] is used to configure metric recording.
type MetricOpts struct {
	Measurement  otelmetric.MeasurementOption
	ResponseData ResponseData
}

// [MetricsRecorder] is an interface for recording metrics.
type MetricsRecorder interface {
	// [RegisterMetric] is called when a metric should be registered.
	RegisterMetric(ctx context.Context, cfg RegisterMetricConfig)

	// [StartMetric] is called when a metric recording should start.
	StartMetric(ctx context.Context, opts MetricOpts)

	// [EndMetric] is called when a metric recording should end.
	// This could be used to record the actual metric.
	EndMetric(ctx context.Context, opts MetricOpts)
}

// [WithMetricRecorders] specifies metric recorders to use for recording metrics.
// If none are specified, no metrics will be recorded.
func WithMetricRecorders(recorders ...MetricsRecorder) Option {
	return optionFunc(func(cfg *config) {
		cfg.MetricRecorders = recorders
	})
}

// WithMeterProvider specifies a meter provider to use for creating a meter.
// If none is specified, the global provider is used.
func WithMeterProvider(provider otelmetric.MeterProvider) Option {
	return optionFunc(func(cfg *config) {
		cfg.MeterProvider = provider
	})
}