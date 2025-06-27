package otelchi_test

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/riandyrn/otelchi"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel"
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
	router.Use(otelchi.Middleware("foobar"))
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
	router.Use(otelchi.Middleware("foobar"))
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
	router.Use(otelchi.Middleware("foobar", otelchi.WithPropagators(prop)))
	router.HandleFunc("/user/{id}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		span := trace.SpanFromContext(r.Context())
		assert.Equal(t, sc, span.SpanContext())
		w.WriteHeader(http.StatusOK)
	}))

	router.ServeHTTP(w, r)
	assert.True(t, called, "failed to run test")
}

func TestWithTracerProvider(t *testing.T) {
	spanRecorder := tracetest.NewSpanRecorder()
	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)
	tracerProvider.RegisterSpanProcessor(spanRecorder)

	router := chi.NewRouter()
	router.Use(otelchi.Middleware("test-server", otelchi.WithTracerProvider(tracerProvider)))
	router.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("OK"))
		assert.NoError(t, err)
	})

	r := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	// Verify that a span was recorded using the provided TracerProvider
	spans := spanRecorder.Ended()
	assert.Len(t, spans, 1)
	assert.Equal(t, "/test", spans[0].Name())
}

func TestSpanFromRequestContext(t *testing.T) {
	spanRecorder := tracetest.NewSpanRecorder()
	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)
	tracerProvider.RegisterSpanProcessor(spanRecorder)

	router := chi.NewRouter()
	router.Use(otelchi.Middleware("test-server"))
	router.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("OK"))
		assert.NoError(t, err)
	})

	tracer := tracerProvider.Tracer("test-tracer")
	ctx, parentSpan := tracer.Start(context.Background(), "parent-span")
	defer parentSpan.End()

	r := httptest.NewRequest("GET", "/test", nil)
	r = r.WithContext(ctx)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	spans := spanRecorder.Ended()
	// Should have 2 spans: the parent span and the middleware span
	assert.GreaterOrEqual(t, len(spans), 1)

	// Find the middleware span
	var middlewareSpan sdktrace.ReadOnlySpan
	for _, span := range spans {
		if span.Name() == "/test" {
			middlewareSpan = span
			break
		}
	}
	assert.NotNil(t, middlewareSpan)
	assert.Equal(t, parentSpan.SpanContext().TraceID(), middlewareSpan.SpanContext().TraceID())
}

func TestResponseWriterInterfaces(t *testing.T) {
	// make sure the recordingResponseWriter preserves interfaces implemented by the wrapped writer
	router := chi.NewRouter()
	router.Use(otelchi.Middleware("foobar"))
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

func TestSuperfluousWriteHeader(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)

	r := chi.NewRouter()
	r.Use(otelchi.Middleware("foobar"))

	r.Get("/test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte("Hello"))
		w.WriteHeader(http.StatusOK) // This is superfluous trigger
	})

	ts := httptest.NewServer(r)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/test")
	assert.NoError(t, err)
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	assert.NoError(t, err)
	assert.Equal(t, "Hello", string(body))
	assert.Equal(t, http.StatusAccepted, resp.StatusCode)

	logOutput := buf.String()
	assert.False(t, strings.Contains(logOutput, "http: superfluous"))
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
