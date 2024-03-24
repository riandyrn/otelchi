package otelchi_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/riandyrn/otelchi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

func TestSDKIntegration(t *testing.T) {
	// prepare router and span recorder
	router, sr := newSDKTestRouter("foobar", false)

	// define routes
	router.HandleFunc("/user/{id:[0-9]+}", ok)
	router.HandleFunc("/book/{title}", ok)

	// execute requests
	reqs := []*http.Request{
		httptest.NewRequest("GET", "/user/123", nil),
		httptest.NewRequest("GET", "/book/foo", nil),
	}
	executeRequests(router, reqs)

	// get recorded spans
	recordedSpans := sr.Ended()

	// ensure that we have 2 recorded spans
	require.Len(t, recordedSpans, len(reqs))

	// ensure span values
	checkSpans(t, recordedSpans, []spanValueCheck{
		{
			Name: "/user/{id:[0-9]+}",
			Kind: trace.SpanKindServer,
			Attributes: getSemanticAttributes(
				"foobar",
				http.StatusOK,
				"GET",
				"/user/123",
				"/user/{id:[0-9]+}",
			),
		},
		{
			Name: "/book/{title}",
			Kind: trace.SpanKindServer,
			Attributes: getSemanticAttributes(
				"foobar",
				http.StatusOK,
				"GET",
				"/book/foo",
				"/book/{title}",
			),
		},
	})
}

func TestSDKIntegrationWithFilters(t *testing.T) {
	// prepare router and span recorder
	router, sr := newSDKTestRouter("foobar", false, otelchi.WithFilter(func(r *http.Request) bool {
		// if client access /live or /ready, there should be no span
		if r.URL.Path == "/live" || r.URL.Path == "/ready" {
			return false
		}

		// otherwise always return the span
		return true
	}))

	// define router
	router.HandleFunc("/user/{id:[0-9]+}", ok)
	router.HandleFunc("/book/{title}", ok)
	router.HandleFunc("/health", ok)
	router.HandleFunc("/ready", ok)

	// execute requests
	executeRequests(router, []*http.Request{
		httptest.NewRequest("GET", "/user/123", nil),
		httptest.NewRequest("GET", "/book/foo", nil),
		httptest.NewRequest("GET", "/live", nil),
		httptest.NewRequest("GET", "/ready", nil),
	})

	// get recorded spans and ensure the length is 2
	recordedSpans := sr.Ended()
	require.Len(t, recordedSpans, 2)

	// ensure span values
	checkSpans(t, recordedSpans, []spanValueCheck{
		{
			Name: "/user/{id:[0-9]+}",
			Kind: trace.SpanKindServer,
			Attributes: getSemanticAttributes(
				"foobar",
				http.StatusOK,
				"GET",
				"/user/123",
				"/user/{id:[0-9]+}",
			),
		},
		{
			Name: "/book/{title}",
			Kind: trace.SpanKindServer,
			Attributes: getSemanticAttributes(
				"foobar",
				http.StatusOK,
				"GET",
				"/book/foo",
				"/book/{title}",
			),
		},
	})
}

func TestSDKIntegrationWithChiRoutes(t *testing.T) {
	// define router & span recorder
	router, sr := newSDKTestRouter("foobar", true)

	// define route
	router.HandleFunc("/user/{id:[0-9]+}", ok)
	router.HandleFunc("/book/{title}", ok)

	// execute requests
	reqs := []*http.Request{
		httptest.NewRequest("GET", "/user/123", nil),
		httptest.NewRequest("GET", "/book/foo", nil),
	}
	executeRequests(router, reqs)

	// get recorded spans
	recordedSpans := sr.Ended()

	// ensure that we have 2 recorded spans
	require.Len(t, recordedSpans, len(reqs))

	// ensure span values
	checkSpans(t, recordedSpans, []spanValueCheck{
		{
			Name: "/user/{id:[0-9]+}",
			Kind: trace.SpanKindServer,
			Attributes: getSemanticAttributes(
				"foobar",
				http.StatusOK,
				"GET",
				"/user/123",
				"/user/{id:[0-9]+}",
			),
		},
		{
			Name: "/book/{title}",
			Kind: trace.SpanKindServer,
			Attributes: getSemanticAttributes(
				"foobar",
				http.StatusOK,
				"GET",
				"/book/foo",
				"/book/{title}",
			),
		},
	})
}

func TestSDKIntegrationOverrideSpanName(t *testing.T) {
	sr := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider()
	provider.RegisterSpanProcessor(sr)

	router := chi.NewRouter()
	router.Use(
		otelchi.Middleware(
			"foobar",
			otelchi.WithTracerProvider(provider),
			otelchi.WithChiRoutes(router),
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
		otelchi.Middleware(
			"foobar",
			otelchi.WithTracerProvider(provider),
			otelchi.WithRequestMethodInSpanName(true),
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
		assert.Equal(t, got[want.Key], want.Value)
	}
}

func ok(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func newSDKTestRouter(serverName string, withChiRoutes bool, opts ...otelchi.Option) (*chi.Mux, *tracetest.SpanRecorder) {
	spanRecorder := tracetest.NewSpanRecorder()
	tracerProvider := sdktrace.NewTracerProvider()
	tracerProvider.RegisterSpanProcessor(spanRecorder)

	opts = append(opts, otelchi.WithTracerProvider(tracerProvider))

	router := chi.NewRouter()
	if withChiRoutes {
		opts = append(opts, otelchi.WithChiRoutes(router))
	}
	router.Use(otelchi.Middleware(serverName, opts...))

	return router, spanRecorder
}

type spanValueCheck struct {
	Name       string
	Kind       trace.SpanKind
	Attributes []attribute.KeyValue
}

func getSemanticAttributes(serverName string, httpStatusCode int, httpMethod, httpTarget, httpRoute string) []attribute.KeyValue {
	return []attribute.KeyValue{
		attribute.String("http.server_name", serverName),
		attribute.Int("http.status_code", httpStatusCode),
		attribute.String("http.method", httpMethod),
		attribute.String("http.target", httpTarget),
		attribute.String("http.route", httpRoute),
	}
}

func checkSpans(t *testing.T, spans []sdktrace.ReadOnlySpan, valueChecks []spanValueCheck) {
	for i := 0; i < len(spans); i++ {
		span := spans[i]
		valueCheck := valueChecks[i]
		assertSpan(t, span, valueCheck.Name, valueCheck.Kind, valueCheck.Attributes...)
	}
}

func executeRequests(router *chi.Mux, reqs []*http.Request) {
	w := httptest.NewRecorder()
	for _, r := range reqs {
		router.ServeHTTP(w, r)
	}
}
