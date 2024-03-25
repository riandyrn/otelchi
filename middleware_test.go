package otelchi

import (
	"bufio"
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

var sc = trace.NewSpanContext(trace.SpanContextConfig{
	TraceID:    [16]byte{1},
	SpanID:     [8]byte{1},
	Remote:     true,
	TraceFlags: trace.FlagsSampled,
})

func TestPassthroughSpanFromGlobalTracer(t *testing.T) {
	var called bool
	router := chi.NewRouter()
	router.Use(Middleware("foobar"))
	// The default global TracerProvider provides "pass through" spans for any
	// span context in the incoming request context.
	router.HandleFunc("/user/{id}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		got := trace.SpanFromContext(r.Context()).SpanContext()
		assert.Equal(t, sc, got)
		w.WriteHeader(http.StatusOK)
	}))

	r := httptest.NewRequest("GET", "/user/123", nil)
	r = r.WithContext(trace.ContextWithRemoteSpanContext(context.Background(), sc))
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)
	assert.True(t, called, "failed to run test")
}

func TestPropagationWithGlobalPropagators(t *testing.T) {
	defer func(p propagation.TextMapPropagator) {
		otel.SetTextMapPropagator(p)
	}(otel.GetTextMapPropagator())

	prop := propagation.TraceContext{}
	otel.SetTextMapPropagator(prop)

	r := httptest.NewRequest("GET", "/user/123", nil)
	w := httptest.NewRecorder()

	ctx := trace.ContextWithRemoteSpanContext(context.Background(), sc)
	otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(r.Header))

	var called bool
	router := chi.NewRouter()
	router.Use(Middleware("foobar"))
	router.HandleFunc("/user/{id}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		span := trace.SpanFromContext(r.Context())
		assert.Equal(t, sc, span.SpanContext())
		w.WriteHeader(http.StatusOK)
	}))

	router.ServeHTTP(w, r)
	assert.True(t, called, "failed to run test")
}

func TestPropagationWithCustomPropagators(t *testing.T) {
	prop := propagation.TraceContext{}

	r := httptest.NewRequest("GET", "/user/123", nil)
	w := httptest.NewRecorder()

	ctx := trace.ContextWithRemoteSpanContext(context.Background(), sc)
	prop.Inject(ctx, propagation.HeaderCarrier(r.Header))

	var called bool
	router := chi.NewRouter()
	router.Use(Middleware("foobar", WithPropagators(prop)))
	router.HandleFunc("/user/{id}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		span := trace.SpanFromContext(r.Context())
		assert.Equal(t, sc, span.SpanContext())
		w.WriteHeader(http.StatusOK)
	}))

	router.ServeHTTP(w, r)
	assert.True(t, called, "failed to run test")
}

type testResponseWriter struct {
	writer http.ResponseWriter
}

func (rw *testResponseWriter) Header() http.Header {
	return rw.writer.Header()
}
func (rw *testResponseWriter) Write(b []byte) (int, error) {
	return rw.writer.Write(b)
}
func (rw *testResponseWriter) WriteHeader(statusCode int) {
	rw.writer.WriteHeader(statusCode)
}

// implement Hijacker
func (rw *testResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return nil, nil, nil
}

// implement Pusher
func (rw *testResponseWriter) Push(target string, opts *http.PushOptions) error {
	return nil
}

// implement Flusher
func (rw *testResponseWriter) Flush() {
}

// implement io.ReaderFrom
func (rw *testResponseWriter) ReadFrom(r io.Reader) (n int64, err error) {
	return 0, nil
}

func TestResponseWriterInterfaces(t *testing.T) {
	// make sure the recordingResponseWriter preserves interfaces implemented by the wrapped writer
	router := chi.NewRouter()
	router.Use(Middleware("foobar"))
	router.HandleFunc("/user/{id}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Implements(t, (*http.Hijacker)(nil), w)
		assert.Implements(t, (*http.Pusher)(nil), w)
		assert.Implements(t, (*http.Flusher)(nil), w)
		assert.Implements(t, (*io.ReaderFrom)(nil), w)
		w.WriteHeader(http.StatusOK)
	}))

	r := httptest.NewRequest("GET", "/user/123", nil)
	w := &testResponseWriter{
		writer: httptest.NewRecorder(),
	}

	router.ServeHTTP(w, r)
}

func ok(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func TestSDKIntegration(t *testing.T) {
	sr := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider()
	provider.RegisterSpanProcessor(sr)

	router := chi.NewRouter()
	router.Use(Middleware("foobar", WithTracerProvider(provider)))
	router.HandleFunc("/user/{id:[0-9]+}", ok)
	router.HandleFunc("/book/{title}", ok)

	r0 := httptest.NewRequest("GET", "/user/123", nil)
	r1 := httptest.NewRequest("GET", "/book/foo", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r0)
	router.ServeHTTP(w, r1)

	require.Len(t, sr.Ended(), 2)
	assertSpan(t, sr.Ended()[0],
		"/user/{id:[0-9]+}",
		trace.SpanKindServer,
		attribute.String("http.server_name", "foobar"),
		attribute.Int("http.status_code", http.StatusOK),
		attribute.String("http.method", "GET"),
		attribute.String("http.target", "/user/123"),
		attribute.String("http.route", "/user/{id:[0-9]+}"),
	)
	require.Equal(t, w.Header().Get("X-Trace-ID"), sr.Ended()[0].SpanContext().TraceID().String())
	assertSpan(t, sr.Ended()[1],
		"/book/{title}",
		trace.SpanKindServer,
		attribute.String("http.server_name", "foobar"),
		attribute.Int("http.status_code", http.StatusOK),
		attribute.String("http.method", "GET"),
		attribute.String("http.target", "/book/foo"),
		attribute.String("http.route", "/book/{title}"),
	)
	require.Equal(t, w.Header().Get("X-Trace-ID"), sr.Ended()[0].SpanContext().TraceID().String())
}

func TestSDKIntegrationWithOverrideHeaderKey(t *testing.T) {
	sr := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider()
	provider.RegisterSpanProcessor(sr)

	customeHeaderKey := "X-Custom-Trace-ID"

	router := chi.NewRouter()
	router.Use(Middleware(
		"foobar",
		WithTracerProvider(provider),
		WithTraceResponseHeaderKey(customeHeaderKey),
	))
	router.HandleFunc("/user/{id:[0-9]+}", ok)
	router.HandleFunc("/book/{title}", ok)

	r0 := httptest.NewRequest("GET", "/user/123", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r0)

	require.Len(t, sr.Ended(), 1)
	assertSpan(t, sr.Ended()[0],
		"/user/{id:[0-9]+}",
		trace.SpanKindServer,
		attribute.String("http.server_name", "foobar"),
		attribute.Int("http.status_code", http.StatusOK),
		attribute.String("http.method", "GET"),
		attribute.String("http.target", "/user/123"),
		attribute.String("http.route", "/user/{id:[0-9]+}"),
	)
	require.Equal(t, w.Header().Get(customeHeaderKey), sr.Ended()[0].SpanContext().TraceID().String())
}

func TestSDKIntegrationWithFilters(t *testing.T) {
	sr := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider()
	provider.RegisterSpanProcessor(sr)

	router := chi.NewRouter()
	router.Use(Middleware("foobar", WithTracerProvider(provider), WithFilter(func(r *http.Request) bool {
		if r.URL.Path == "/live" || r.URL.Path == "/ready" {
			return false
		}
		return true
	})))
	router.HandleFunc("/user/{id:[0-9]+}", ok)
	router.HandleFunc("/book/{title}", ok)
	router.HandleFunc("/health", ok)
	router.HandleFunc("/ready", ok)

	r0 := httptest.NewRequest("GET", "/user/123", nil)
	r1 := httptest.NewRequest("GET", "/book/foo", nil)
	r2 := httptest.NewRequest("GET", "/live", nil)
	r3 := httptest.NewRequest("GET", "/ready", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r0)
	router.ServeHTTP(w, r1)
	router.ServeHTTP(w, r2)
	router.ServeHTTP(w, r3)

	require.Len(t, sr.Ended(), 2)
	assertSpan(t, sr.Ended()[0],
		"/user/{id:[0-9]+}",
		trace.SpanKindServer,
		attribute.String("http.server_name", "foobar"),
		attribute.Int("http.status_code", http.StatusOK),
		attribute.String("http.method", "GET"),
		attribute.String("http.target", "/user/123"),
		attribute.String("http.route", "/user/{id:[0-9]+}"),
	)
	assertSpan(t, sr.Ended()[1],
		"/book/{title}",
		trace.SpanKindServer,
		attribute.String("http.server_name", "foobar"),
		attribute.Int("http.status_code", http.StatusOK),
		attribute.String("http.method", "GET"),
		attribute.String("http.target", "/book/foo"),
		attribute.String("http.route", "/book/{title}"),
	)
}

func TestSDKIntegrationWithChiRoutes(t *testing.T) {
	sr := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider()
	provider.RegisterSpanProcessor(sr)

	router := chi.NewRouter()
	router.Use(
		Middleware(
			"foobar",
			WithTracerProvider(provider),
			WithChiRoutes(router),
		),
	)
	router.HandleFunc("/user/{id:[0-9]+}", ok)
	router.HandleFunc("/book/{title}", ok)

	r0 := httptest.NewRequest("GET", "/user/123", nil)
	r1 := httptest.NewRequest("GET", "/book/foo", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r0)
	router.ServeHTTP(w, r1)

	require.Len(t, sr.Ended(), 2)
	assertSpan(t, sr.Ended()[0],
		"/user/{id:[0-9]+}",
		trace.SpanKindServer,
		attribute.String("http.server_name", "foobar"),
		attribute.Int("http.status_code", http.StatusOK),
		attribute.String("http.method", "GET"),
		attribute.String("http.target", "/user/123"),
		attribute.String("http.route", "/user/{id:[0-9]+}"),
	)
	assertSpan(t, sr.Ended()[1],
		"/book/{title}",
		trace.SpanKindServer,
		attribute.String("http.server_name", "foobar"),
		attribute.Int("http.status_code", http.StatusOK),
		attribute.String("http.method", "GET"),
		attribute.String("http.target", "/book/foo"),
		attribute.String("http.route", "/book/{title}"),
	)
}

func TestSDKIntegrationOverrideSpanName(t *testing.T) {
	sr := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider()
	provider.RegisterSpanProcessor(sr)

	router := chi.NewRouter()
	router.Use(
		Middleware(
			"foobar",
			WithTracerProvider(provider),
			WithChiRoutes(router),
		),
	)
	router.HandleFunc("/user/{id:[0-9]+}", func(w http.ResponseWriter, r *http.Request) {
		span := trace.SpanFromContext(r.Context())
		span.SetName("overriden span name")
		w.WriteHeader(http.StatusOK)
	})
	router.HandleFunc("/book/{title}", ok)

	r0 := httptest.NewRequest("GET", "/user/123", nil)
	r1 := httptest.NewRequest("GET", "/book/foo", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r0)
	router.ServeHTTP(w, r1)

	require.Len(t, sr.Ended(), 2)
	assertSpan(t, sr.Ended()[0],
		"overriden span name",
		trace.SpanKindServer,
		attribute.String("http.server_name", "foobar"),
		attribute.Int("http.status_code", http.StatusOK),
		attribute.String("http.method", "GET"),
		attribute.String("http.target", "/user/123"),
		attribute.String("http.route", "/user/{id:[0-9]+}"),
	)
	assertSpan(t, sr.Ended()[1],
		"/book/{title}",
		trace.SpanKindServer,
		attribute.String("http.server_name", "foobar"),
		attribute.Int("http.status_code", http.StatusOK),
		attribute.String("http.method", "GET"),
		attribute.String("http.target", "/book/foo"),
		attribute.String("http.route", "/book/{title}"),
	)
}

func TestSDKIntegrationWithRequestMethodInSpanName(t *testing.T) {
	sr := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider()
	provider.RegisterSpanProcessor(sr)

	router := chi.NewRouter()
	router.Use(
		Middleware(
			"foobar",
			WithTracerProvider(provider),
			WithRequestMethodInSpanName(true),
		),
	)
	router.HandleFunc("/user/{id:[0-9]+}", ok)
	router.HandleFunc("/book/{title}", ok)

	r0 := httptest.NewRequest("GET", "/user/123", nil)
	r1 := httptest.NewRequest("GET", "/book/foo", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r0)
	router.ServeHTTP(w, r1)

	require.Len(t, sr.Ended(), 2)
	assertSpan(t, sr.Ended()[0],
		"GET /user/{id:[0-9]+}",
		trace.SpanKindServer,
		attribute.String("http.server_name", "foobar"),
		attribute.Int("http.status_code", http.StatusOK),
		attribute.String("http.method", "GET"),
		attribute.String("http.target", "/user/123"),
		attribute.String("http.route", "/user/{id:[0-9]+}"),
	)
	assertSpan(t, sr.Ended()[1],
		"GET /book/{title}",
		trace.SpanKindServer,
		attribute.String("http.server_name", "foobar"),
		attribute.Int("http.status_code", http.StatusOK),
		attribute.String("http.method", "GET"),
		attribute.String("http.target", "/book/foo"),
		attribute.String("http.route", "/book/{title}"),
	)
}

func TestEmptyHandlerDefaultStatusCode(t *testing.T) {
	sr := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider()
	provider.RegisterSpanProcessor(sr)

	router := chi.NewRouter()
	router.Use(
		Middleware(
			"foobar",
			WithTracerProvider(provider),
			WithRequestMethodInSpanName(true),
		),
	)
	router.HandleFunc("/", func(_ http.ResponseWriter, _ *http.Request) {})
	router.HandleFunc("/not_found", http.NotFound)
	r := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)
	assert.Equal(t, http.StatusOK, w.Result().StatusCode)

	r = httptest.NewRequest("GET", "/not_found", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, r)
	assert.Equal(t, http.StatusNotFound, w.Result().StatusCode)

	require.Len(t, sr.Ended(), 2)
	assertSpan(t, sr.Ended()[0],
		"GET /",
		trace.SpanKindServer,
		attribute.String("http.server_name", "foobar"),
		attribute.Int("http.status_code", http.StatusOK),
		attribute.String("http.method", "GET"),
		attribute.String("http.target", "/"),
		attribute.String("http.route", ""),
	)

	assertSpan(t, sr.Ended()[1],
		"GET /not_found",
		trace.SpanKindServer,
		attribute.String("http.server_name", "foobar"),
		attribute.Int("http.status_code", http.StatusNotFound),
		attribute.String("http.method", "GET"),
		attribute.String("http.target", "/not_found"),
		attribute.String("http.route", "/not_found"),
	)
}

func assertSpan(t *testing.T, span sdktrace.ReadOnlySpan, name string, kind trace.SpanKind, attrs ...attribute.KeyValue) {
	assert.Equal(t, name, span.Name())
	assert.Equal(t, kind, span.SpanKind())

	got := make(map[attribute.Key]attribute.Value, len(span.Attributes()))
	for _, a := range span.Attributes() {
		got[a.Key] = a.Value
	}
	for _, want := range attrs {
		if !assert.Contains(t, got, want.Key) {
			continue
		}
		assert.Equal(t, want.Value, got[want.Key])
	}
}
