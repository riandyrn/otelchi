package otelchi

import (
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/felixge/httpsnoop"
	"github.com/go-chi/chi/v5"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"

	otelcontrib "go.opentelemetry.io/contrib"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	oteltrace "go.opentelemetry.io/otel/trace"
)

const (
	tracerName = "github.com/riandyrn/otelchi"
)

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
		oteltrace.WithInstrumentationVersion(otelcontrib.SemVersion()),
	)
	if cfg.Propagators == nil {
		cfg.Propagators = otel.GetTextMapPropagator()
	}
	return func(handler http.Handler) http.Handler {
		return traceware{
			serverName:          serverName,
			tracer:              tracer,
			propagators:         cfg.Propagators,
			handler:             handler,
			chiRoutes:           cfg.ChiRoutes,
			reqMethodInSpanName: cfg.RequestMethodInSpanName,
			filter:              cfg.Filter,
		}
	}
}

type traceware struct {
	serverName          string
	tracer              oteltrace.Tracer
	propagators         propagation.TextMapPropagator
	handler             http.Handler
	chiRoutes           chi.Routes
	reqMethodInSpanName bool
	filter              func(r *http.Request) bool
}

type recordingResponseWriter struct {
	writer  http.ResponseWriter
	written bool
	status  int
	length  int
}

var rrwPool = &sync.Pool{
	New: func() interface{} {
		return &recordingResponseWriter{}
	},
}

func getRRW(writer http.ResponseWriter) *recordingResponseWriter {
	rrw := rrwPool.Get().(*recordingResponseWriter)
	rrw.written = false
	rrw.status = 0
	rrw.writer = httpsnoop.Wrap(writer, httpsnoop.Hooks{
		Write: func(next httpsnoop.WriteFunc) httpsnoop.WriteFunc {
			return func(b []byte) (int, error) {
				if !rrw.written {
					rrw.written = true
					rrw.status = http.StatusOK
					rrw.length = len(b)
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

func getHostAndPort(r *http.Request) (string, int) {
	parts := strings.Split(r.Host, ":")

	if (len(parts) > 1) && (len(parts[1]) > 0) {
		port, err := strconv.Atoi(parts[1])
		if err != nil {
			panic(err)
		}
		return parts[0], port
	}
	return parts[0], 80
}

// ServeHTTP implements the http.Handler interface. It does the actual
// tracing of the request.
func (tw traceware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// skip if filter returns false
	if tw.filter != nil && !tw.filter(r) {
		tw.handler.ServeHTTP(w, r)
		return
	}

	// extract tracing header using propagator
	ctx := tw.propagators.Extract(r.Context(), propagation.HeaderCarrier(r.Header))
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
	if tw.chiRoutes != nil {
		rctx := chi.NewRouteContext()
		if tw.chiRoutes.Match(rctx, r.Method, r.URL.Path) {
			routePattern = rctx.RoutePattern()
			spanName = addPrefixToSpanName(tw.reqMethodInSpanName, r.Method, routePattern)
		}
	}

	// Request information gathering
	// Host and port
	host, port := getHostAndPort(r)
	var peer_port int
	var err error
	if len(r.URL.Port()) > 0 {
		peer_port, err = strconv.Atoi(r.URL.Port())
		if err != nil {
			panic(err)
		}
	}

	// Request body length
	length, err := io.Copy(io.Discard, r.Body)

	if err != nil {
		panic(err)
	}

	ctx, span := tw.tracer.Start(
		ctx, spanName,
		// oteltrace.WithAttributes(semconv.NetAttributesFromHTTPRequest("tcp", r)...),
		oteltrace.WithAttributes(semconv.ServerAddress(host)),
		oteltrace.WithAttributes(semconv.ServerPort(port)),
		oteltrace.WithAttributes(semconv.NetworkProtocolName("http")),
		oteltrace.WithAttributes(semconv.NetworkProtocolVersion(r.Proto)),
		oteltrace.WithAttributes(semconv.NetworkPeerAddress(r.RemoteAddr)),
		oteltrace.WithAttributes(semconv.NetworkPeerAddress(r.RemoteAddr)),
		oteltrace.WithAttributes(semconv.NetworkPeerPort(peer_port)),
		oteltrace.WithAttributes(semconv.URLPath(r.URL.Path)),
		oteltrace.WithAttributes(semconv.URLQuery(r.URL.RawQuery)),
		oteltrace.WithAttributes(semconv.URLScheme(r.URL.Scheme)),
		oteltrace.WithAttributes(semconv.ClientAddress(r.RemoteAddr)),
		oteltrace.WithAttributes(semconv.HTTPRequestMethodKey.String(r.Method)),
		oteltrace.WithAttributes(semconv.HTTPRequestBodySize(int(length))),
		oteltrace.WithSpanKind(oteltrace.SpanKindServer),
	)
	defer span.End()

	// get recording response writer
	rrw := getRRW(w)
	defer putRRW(rrw)

	// execute next http handler
	r = r.WithContext(ctx)
	tw.handler.ServeHTTP(rrw.writer, r)

	// set span name & http route attribute if necessary
	if len(routePattern) == 0 {
		routePattern = chi.RouteContext(r.Context()).RoutePattern()
		span.SetAttributes(semconv.URLFull(routePattern))
		span.SetAttributes(semconv.HTTPResponseBodySize(rrw.length))

		spanName = addPrefixToSpanName(tw.reqMethodInSpanName, r.Method, routePattern)
		span.SetName(spanName)
	}

	// set status code attribute
	span.SetAttributes(semconv.HTTPResponseStatusCodeKey.Int(rrw.status))

	// set span status FIXME
	// spanStatus, spanMessage := semconv.SpanStatusFromHTTPStatusCode(rrw.status)
	span.SetStatus(codes.Code(rrw.status), "")
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
