package metrics

import (
	otelmetric "go.opentelemetry.io/otel/metric"
)

const (
	ScopeName = "github.com/riandyrn/otelchi/metrics"
)

// config is used to configure the metrics middleware.
type config struct {
	MeterProvider otelmetric.MeterProvider
}

// Option specifies instrumentation configuration options.
type Option interface {
	apply(*config)
}

type optionFunc func(*config)

func (o optionFunc) apply(c *config) {
	o(c)
}

// WithMeterProvider specifies a meter provider to use for creating a meter.
// If none is specified, the global provider is used.
func WithMeterProvider(provider otelmetric.MeterProvider) Option {
	return optionFunc(func(cfg *config) {
		cfg.MeterProvider = provider
	})
}
