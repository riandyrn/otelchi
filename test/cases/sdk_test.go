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
	// prepare test router and span recorder
	router, sr := newSDKTestRouter("foobar", true)

	// define route
	router.HandleFunc("/user/{id:[0-9]+}", func(w http.ResponseWriter, r *http.Request) {
		span := trace.SpanFromContext(r.Context())
		span.SetName("overriden span name")
		w.WriteHeader(http.StatusOK)
	})
	router.HandleFunc("/book/{title}", ok)

	// execute requests
	reqs := []*http.Request{
		httptest.NewRequest("GET", "/user/123", nil),
		httptest.NewRequest("GET", "/book/foo", nil),
	}
	executeRequests(router, reqs)

	// get recorded spans
	recordedSpans := sr.Ended()

	// ensure the number of spans is correct
	require.Len(t, sr.Ended(), len(reqs))

	// check span values
	checkSpans(t, recordedSpans, []spanValueCheck{
		{
			Name: "overriden span name",
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

func TestSDKIntegrationWithRequestMethodInSpanName(t *testing.T) {
	// prepare router & span recorder
	router, sr := newSDKTestRouter("foobar", true, otelchi.WithRequestMethodInSpanName(true))

	// define handler
	router.HandleFunc("/user/{id:[0-9]+}", ok)
	router.HandleFunc("/book/{title}", ok)

	// execute requests
	reqs := []*http.Request{
		httptest.NewRequest("GET", "/user/123", nil),
		httptest.NewRequest("GET", "/book/foo", nil),
	}
	executeRequests(router, reqs)

	// get recorded spans & ensure the number is correct
	recordedSpans := sr.Ended()
	require.Len(t, sr.Ended(), len(reqs))

	// check span values
	checkSpans(t, recordedSpans, []spanValueCheck{
		{
			Name: "GET /user/{id:[0-9]+}",
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
			Name: "GET /book/{title}",
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

func TestSDKIntegrationEmptyHandlerDefaultStatusCode(t *testing.T) {
	// prepare router and span recorder
	router, sr := newSDKTestRouter("foobar", false)

	// define routes
	router.HandleFunc("/user/{id:[0-9]+}", func(w http.ResponseWriter, r *http.Request) {})
	router.HandleFunc("/book/{title}", func(w http.ResponseWriter, r *http.Request) {})
	router.HandleFunc("/not-found", http.NotFound)

	// execute requests
	reqs := []*http.Request{
		httptest.NewRequest("GET", "/user/123", nil),
		httptest.NewRequest("GET", "/book/foo", nil),
		httptest.NewRequest("GET", "/not-found", nil),
	}
	executeRequests(router, reqs)

	// get recorded spans
	recordedSpans := sr.Ended()

	// ensure that we have 3 recorded spans
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
		{
			Name: "/not-found",
			Kind: trace.SpanKindServer,
			Attributes: getSemanticAttributes(
				"foobar",
				http.StatusNotFound,
				"GET",
				"/not-found",
				"/not-found",
			),
		},
	})
}

func TestSDKIntegrationRootHandler(t *testing.T) {
	// prepare router and span recorder
	router, sr := newSDKTestRouter("foobar", true)

	// define routes
	router.HandleFunc("/", ok)

	// execute requests
	reqs := []*http.Request{
		httptest.NewRequest("GET", "/", nil),
	}
	executeRequests(router, reqs)

	// get recorded spans
	recordedSpans := sr.Ended()

	// ensure that we have 1 recorded span
	require.Len(t, recordedSpans, 1)

	// ensure span values
	checkSpans(t, recordedSpans, []spanValueCheck{
		{
			Name: "/",
			Kind: trace.SpanKindServer,
			Attributes: getSemanticAttributes(
				"foobar",
				http.StatusOK,
				"GET",
				"/",
				"/",
			),
		},
	})
}

func TestSDKIntegrationWithOverrideHeaderKey(t *testing.T) {
	sr := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider()
	provider.RegisterSpanProcessor(sr)

	customeHeaderKey := "X-Custom-Trace-ID"

	router := chi.NewRouter()
	router.Use(otelchi.Middleware(
		"foobar",
		otelchi.WithTracerProvider(provider),
		otelchi.WithTraceResponseHeaderKey(customeHeaderKey),
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
