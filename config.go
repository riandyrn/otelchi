package otelchi

import (
	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/otel/propagation"
	oteltrace "go.opentelemetry.io/otel/trace"
)

// config is used to configure the mux middleware.
type config struct {
	TracerProvider oteltrace.TracerProvider
	Propagators    propagation.TextMapPropagator
	ChiRoutes      chi.Routes
}

// Option specifies instrumentation configuration options.
type Option interface {
	apply(*config)
}

type optionFunc func(*config)

func (o optionFunc) apply(c *config) {
	o(c)
}

// WithPropagators specifies propagators to use for extracting
// information from the HTTP requests. If none are specified, global
// ones will be used.
func WithPropagators(propagators propagation.TextMapPropagator) Option {
	return optionFunc(func(cfg *config) {
		cfg.Propagators = propagators
	})
}

// WithTracerProvider specifies a tracer provider to use for creating a tracer.
// If none is specified, the global provider is used.
func WithTracerProvider(provider oteltrace.TracerProvider) Option {
	return optionFunc(func(cfg *config) {
		cfg.TracerProvider = provider
	})
}

// WithChiRoutes specified the routes that being used by application. Its main
// purpose is to provide route pattern on span creation.
func WithChiRoutes(routes chi.Routes) Option {
	return optionFunc(func(cfg *config) {
		cfg.ChiRoutes = routes
	})
}
