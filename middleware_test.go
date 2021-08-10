package otelchi_test

import (
	"bufio"
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi"
	"github.com/riandyrn/otelchi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/oteltest"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"

	b3prop "go.opentelemetry.io/contrib/propagators/b3"
	oteltrace "go.opentelemetry.io/otel/trace"
)

func TestChildSpanFromGlobalTracer(t *testing.T) {
	otel.SetTracerProvider(oteltest.NewTracerProvider())

	router := chi.NewRouter()
	router.Use(otelchi.Middleware("foobar"))
	router.HandleFunc("/user/{id}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		span := oteltrace.SpanFromContext(r.Context())
		_, ok := span.(*oteltest.Span)
		assert.True(t, ok)
		w.WriteHeader(http.StatusOK)
	}))

	r := httptest.NewRequest(http.MethodGet, "/user/123", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)
}

func TestChildSpanFromCustomTracer(t *testing.T) {
	customProvider := oteltest.NewTracerProvider()

	router := chi.NewRouter()
	router.Use(otelchi.Middleware("foobar", otelchi.WithTracerProvider(customProvider)))
	router.HandleFunc("/user/{id}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		span := oteltrace.SpanFromContext(r.Context())
		_, ok := span.(*oteltest.Span)
		assert.True(t, ok)
		w.WriteHeader(http.StatusOK)
	}))

	r := httptest.NewRequest(http.MethodGet, "/user/123", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)
}

func TestChildSpanAttributes(t *testing.T) {
	testCases := []struct {
		Name              string
		Request           *http.Request
		SpanName          string
		HTTPMethod        string
		Target            string
		Route             string
		RespContentLength int
	}{
		{
			Name:              "Test First Route",
			Request:           httptest.NewRequest(http.MethodGet, "/user/123?query=abc", nil),
			SpanName:          "/user/{id}",
			HTTPMethod:        http.MethodGet,
			Target:            "/user/123?query=abc",
			Route:             "/user/{id}",
			RespContentLength: 0,
		},
		{
			Name:              "Test Second Route",
			Request:           httptest.NewRequest(http.MethodPost, "/book/hello_world?output=json", nil),
			SpanName:          "/book/{title}",
			HTTPMethod:        http.MethodPost,
			Target:            "/book/hello_world?output=json",
			Route:             "/book/{title}",
			RespContentLength: 2,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			spanRecorder := new(oteltest.SpanRecorder)
			provider := oteltest.NewTracerProvider(oteltest.WithSpanRecorder(spanRecorder))

			router := chi.NewRouter()
			router.Use(otelchi.Middleware("foobar", otelchi.WithTracerProvider(provider)))
			router.HandleFunc(
				"/user/{id}",
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
				}),
			)
			router.HandleFunc(
				"/book/{title}",
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Write([]byte("ok"))
				}),
			)

			w := httptest.NewRecorder()
			router.ServeHTTP(w, testCase.Request)

			spans := spanRecorder.Completed()
			require.Len(t, spans, 1)

			span := spans[0]
			attributeMap := span.Attributes()
			assert.Equal(t, testCase.SpanName, span.Name())
			assert.Equal(t, oteltrace.SpanKindServer, span.SpanKind())
			assert.Equal(t, attribute.StringValue("foobar"), attributeMap[semconv.HTTPServerNameKey])
			assert.Equal(t, attribute.IntValue(http.StatusOK), attributeMap[semconv.HTTPStatusCodeKey])
			assert.Equal(t, attribute.StringValue(testCase.HTTPMethod), attributeMap[semconv.HTTPMethodKey])
			assert.Equal(t, attribute.StringValue(testCase.Target), attributeMap[semconv.HTTPTargetKey])
			assert.Equal(t, attribute.StringValue(testCase.Route), attributeMap[semconv.HTTPRouteKey])
			assert.Equal(t, attribute.IntValue(testCase.RespContentLength), attributeMap[semconv.HTTPResponseContentLengthKey])
		})
	}
}

func TestGetSpanNotInstrumented(t *testing.T) {
	router := chi.NewRouter()
	router.HandleFunc("/user/{id}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		span := oteltrace.SpanFromContext(r.Context())
		ok := !span.SpanContext().IsValid()
		assert.True(t, ok)
		w.WriteHeader(http.StatusOK)
	}))

	r := httptest.NewRequest("GET", "/user/123", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)
}

func TestPropagationWithGlobalPropagators(t *testing.T) {
	sr := new(oteltest.SpanRecorder)
	provider := oteltest.NewTracerProvider(oteltest.WithSpanRecorder(sr))
	otel.SetTextMapPropagator(propagation.TraceContext{})

	r := httptest.NewRequest("GET", "/user/123", nil)
	w := httptest.NewRecorder()

	ctx, pspan := provider.Tracer(otelchi.TracerName).Start(context.Background(), "test")
	otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(r.Header))

	router := chi.NewRouter()
	router.Use(otelchi.Middleware("foobar", otelchi.WithTracerProvider(provider)))
	router.HandleFunc("/user/{id}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		span := oteltrace.SpanFromContext(r.Context())
		mspan, ok := span.(*oteltest.Span)
		require.True(t, ok)
		assert.Equal(t, pspan.SpanContext().TraceID(), mspan.SpanContext().TraceID())
		assert.Equal(t, pspan.SpanContext().SpanID(), mspan.ParentSpanID())
		w.WriteHeader(http.StatusOK)
	}))

	router.ServeHTTP(w, r)
}

func TestPropagationWithCustomPropagators(t *testing.T) {
	sr := new(oteltest.SpanRecorder)
	provider := oteltest.NewTracerProvider(oteltest.WithSpanRecorder(sr))

	b3 := b3prop.New()

	r := httptest.NewRequest("GET", "/user/123", nil)
	w := httptest.NewRecorder()

	ctx, pspan := provider.Tracer(otelchi.TracerName).Start(context.Background(), "test")
	b3.Inject(ctx, propagation.HeaderCarrier(r.Header))

	router := chi.NewRouter()
	router.Use(otelchi.Middleware("foobar", otelchi.WithTracerProvider(provider), otelchi.WithPropagators(b3)))
	router.HandleFunc("/user/{id}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		span := oteltrace.SpanFromContext(r.Context())
		mspan, ok := span.(*oteltest.Span)
		require.True(t, ok)
		assert.Equal(t, pspan.SpanContext().TraceID(), mspan.SpanContext().TraceID())
		assert.Equal(t, pspan.SpanContext().SpanID(), mspan.ParentSpanID())
		w.WriteHeader(http.StatusOK)
	}))

	router.ServeHTTP(w, r)
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
	provider := oteltest.NewTracerProvider()

	router := chi.NewRouter()
	router.Use(otelchi.Middleware("foobar", otelchi.WithTracerProvider(provider)))
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
