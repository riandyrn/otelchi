package otelchi

import (
	"net/http"
	"sync"
	"time"

	"github.com/felixge/httpsnoop"
	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/contrib"
	"go.opentelemetry.io/otel"
	otelmetric "go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/semconv/v1.12.0"
	oteltrace "go.opentelemetry.io/otel/trace"
)

const tracerName = "github.com/riandyrn/otelchi"

// Middleware sets up a handler to start tracing the incoming
// requests. The serverName parameter should describe the name of the
// (virtual) server handling the request.
func Middleware(serverName string, opts ...Option) func(next http.Handler) http.Handler {
	cfg := config{}
	for _, opt := range opts {
		opt.apply(&cfg)
	}

	if cfg.TracerProvider == nil {
		cfg.TracerProvider = otel.GetTracerProvider()
	}
	tracer := cfg.TracerProvider.Tracer(
		tracerName,
		oteltrace.WithInstrumentationVersion(contrib.Version()),
	)

	if cfg.MeterProvider == nil {
		cfg.MeterProvider = otel.GetMeterProvider()
	}
	meter := cfg.MeterProvider.Meter(
		tracerName,
		otelmetric.WithInstrumentationVersion(contrib.Version()),
	)
	recorder := newMetricsRecorder(meter)

	if cfg.Propagators == nil {
		cfg.Propagators = otel.GetTextMapPropagator()
	}
	return func(handler http.Handler) http.Handler {
		return &otelware{
			serverName:             serverName,
			tracer:                 tracer,
			meter:                  meter,
			recorder:               recorder,
			propagators:            cfg.Propagators,
			handler:                handler,
			chiRoutes:              cfg.ChiRoutes,
			reqMethodInSpanName:    cfg.RequestMethodInSpanName,
			filter:                 cfg.Filter,
			disableMeasureInflight: cfg.DisableMeasureInflight,
			disableMeasureSize:     cfg.DisableMeasureSize,
		}
	}
}

type otelware struct {
	serverName             string
	tracer                 oteltrace.Tracer
	meter                  otelmetric.Meter
	recorder               *metricsRecorder
	propagators            propagation.TextMapPropagator
	handler                http.Handler
	chiRoutes              chi.Routes
	reqMethodInSpanName    bool
	filter                 func(r *http.Request) bool
	disableMeasureInflight bool
	disableMeasureSize     bool
}

type recordingResponseWriter struct {
	writer       http.ResponseWriter
	written      bool
	writtenBytes int64
	status       int
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
	rrw.status = 0
	rrw.writer = httpsnoop.Wrap(writer, httpsnoop.Hooks{
		Write: func(next httpsnoop.WriteFunc) httpsnoop.WriteFunc {
			return func(b []byte) (int, error) {
				if !rrw.written {
					rrw.written = true
					rrw.writtenBytes += int64(len(b))
					rrw.status = http.StatusOK
				}
				return next(b)
			}
		},
		WriteHeader: func(next httpsnoop.WriteHeaderFunc) httpsnoop.WriteHeaderFunc {
			return func(statusCode int) {
				if !rrw.written {
					rrw.written = true
					rrw.status = statusCode
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

// ServeHTTP implements the http.Handler interface. It does the actual
// tracing of the request.
func (ow *otelware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// skip if filter returns false
	if ow.filter != nil && !ow.filter(r) {
		ow.handler.ServeHTTP(w, r)
		return
	}

	// extract tracing header using propagator
	ctx := ow.propagators.Extract(r.Context(), propagation.HeaderCarrier(r.Header))
	// create span, based on specification, we need to set already known attributes
	// when creating the span, the only thing missing here is HTTP route pattern since
	// in go-chi/chi route pattern could only be extracted once the request is executed
	// check here for details:
	//
	// https://github.com/go-chi/chi/issues/150#issuecomment-278850733
	//
	// if we have access to chi routes, we could extract the route pattern beforehand.
	spanName := ""
	routePattern := ""
	if ow.chiRoutes != nil {
		rctx := chi.NewRouteContext()
		if ow.chiRoutes.Match(rctx, r.Method, r.URL.Path) {
			routePattern = rctx.RoutePattern()
			spanName = addPrefixToSpanName(ow.reqMethodInSpanName, r.Method, routePattern)
		}
	}

	props := httpReqProperties{
		Service: ow.serverName,
		ID:      routePattern,
		Method:  r.Method,
	}
	if routePattern == "" {
		props.ID = r.URL.Path
	}

	if !ow.disableMeasureInflight {
		ow.recorder.RecordRequestsInflight(ctx, props, 1)
		defer ow.recorder.RecordRequestsInflight(ctx, props, -1)
	}

	ctx, span := ow.tracer.Start(
		ctx, spanName,
		oteltrace.WithAttributes(semconv.NetAttributesFromHTTPRequest("tcp", r)...),
		oteltrace.WithAttributes(semconv.EndUserAttributesFromHTTPRequest(r)...),
		oteltrace.WithAttributes(semconv.HTTPServerAttributesFromHTTPRequest(ow.serverName, routePattern, r)...),
		oteltrace.WithSpanKind(oteltrace.SpanKindServer),
	)
	defer span.End()

	// get recording response writer
	rrw := getRRW(w)
	defer putRRW(rrw)

	// execute next http handler
	r = r.WithContext(ctx)
	start := time.Now()
	ow.handler.ServeHTTP(rrw.writer, r)

	duration := time.Since(start)

	props.Code = rrw.status
	ow.recorder.RecordRequestDuration(ctx, props, duration)

	if !ow.disableMeasureSize {
		ow.recorder.RecordResponseSize(ctx, props, rrw.writtenBytes)
	}

	// set span name & http route attribute if necessary
	if len(routePattern) == 0 {
		routePattern = chi.RouteContext(r.Context()).RoutePattern()
		span.SetAttributes(semconv.HTTPRouteKey.String(routePattern))

		spanName = addPrefixToSpanName(ow.reqMethodInSpanName, r.Method, routePattern)
		span.SetName(spanName)
	}

	if rrw.status > 0 {
		// set status code attribute
		span.SetAttributes(semconv.HTTPStatusCodeKey.Int(rrw.status))
	}

	// set span status
	spanStatus, spanMessage := semconv.SpanStatusFromHTTPStatusCode(rrw.status)
	span.SetStatus(spanStatus, spanMessage)
}

func addPrefixToSpanName(shouldAdd bool, prefix, spanName string) string {
	// in chi v5.0.8, the root route will be returned has an empty string
	// (see github.com/go-chi/chi/v5@v5.0.8/context.go:126)
	if spanName == "" {
		spanName = "/"
	}

	if shouldAdd && len(spanName) > 0 {
		spanName = prefix + " " + spanName
	}
	return spanName
}
