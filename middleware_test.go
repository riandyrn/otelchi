package otelchi_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi"
	"github.com/riandyrn/otelchi"
	"github.com/stretchr/testify/require"
	"github.com/tj/assert"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/oteltest"

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

func TestChildSpanNames(t *testing.T) {
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

	r, _ := http.NewRequest(http.MethodGet, "/user/123?query=abc", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	spans := spanRecorder.Completed()
	require.Len(t, spans, 1)

	span := spans[0]
	assert.Equal(t, "/user/{id}", span.Name())
	assert.Equal(t, oteltrace.SpanKindServer, span.SpanKind())
	assert.Equal(t, attribute.StringValue("foobar"), span.Attributes()["http.server_name"])
	assert.Equal(t, attribute.IntValue(http.StatusOK), span.Attributes()["http.status_code"])
	assert.Equal(t, attribute.StringValue("GET"), span.Attributes()["http.method"])
	assert.Equal(t, attribute.StringValue("/user/123?query=abc"), span.Attributes()["http.target"])
	assert.Equal(t, attribute.StringValue("/user/{id}"), span.Attributes()["http.route"])
}

func TestPropagationWithGlobalPropagators(t *testing.T) {}

func TestPropagationWithCustomPropagators(t *testing.T) {}

func TestResponseWriterInterfaces(t *testing.T) {}
