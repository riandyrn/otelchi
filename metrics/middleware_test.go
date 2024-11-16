package metrics_test

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
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/noop"

	"github.com/riandyrn/otelchi/metrics"
)

type mockMetricsRecorder struct {
	startCalled     bool
	endCalled       bool
	registerCalled  bool
	meter           metric.Meter
	panicOnRegister bool
}

func (m *mockMetricsRecorder) RegisterMetric(ctx context.Context, cfg metrics.RegisterMetricConfig) {
	if m.panicOnRegister {
		panic("intentional panic during registration")
	}
	m.registerCalled = true
	m.meter = cfg.Meter
}

func (m *mockMetricsRecorder) StartMetric(ctx context.Context, opts metrics.MetricOpts) {
	m.startCalled = true
}

func (m *mockMetricsRecorder) EndMetric(ctx context.Context, opts metrics.MetricOpts) {
	m.endCalled = true
}

// Custom response writer that implements additional interfaces
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

func (rw *testResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return nil, nil, nil
}

func (rw *testResponseWriter) Flush() {}

func (rw *testResponseWriter) ReadFrom(r io.Reader) (n int64, err error) {
	return 0, nil
}

func (rw *testResponseWriter) Push(target string, opts *http.PushOptions) error {
	return nil
}

func TestResponseWriterInterfaces(t *testing.T) {
	recorder := &mockMetricsRecorder{}

	router := chi.NewRouter()
	router.Use(metrics.Middleware("test-server", metrics.WithMetricRecorders(recorder)))
	router.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		// Test that our wrapped ResponseWriter implements all the required interfaces
		assert.Implements(t, (*http.Hijacker)(nil), w)
		assert.Implements(t, (*http.Flusher)(nil), w)
		assert.Implements(t, (*io.ReaderFrom)(nil), w)
		assert.Implements(t, (*http.Pusher)(nil), w)
		w.WriteHeader(http.StatusOK)
	})

	r := httptest.NewRequest("GET", "/test", nil)
	w := &testResponseWriter{
		writer: httptest.NewRecorder(),
	}

	router.ServeHTTP(w, r)
}

func TestGlobalMeterProvider(t *testing.T) {
	// Save the current global MeterProvider and restore it after the test
	originalProvider := otel.GetMeterProvider()
	defer otel.SetMeterProvider(originalProvider)

	// Set a custom global MeterProvider
	customProvider := noop.NewMeterProvider()
	otel.SetMeterProvider(customProvider)

	recorder := &mockMetricsRecorder{}

	router := chi.NewRouter()
	router.Use(metrics.Middleware(
		"test-server",
		metrics.WithMetricRecorders(recorder),
	))
	router.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	r := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	assert.True(t, recorder.registerCalled)
	assert.True(t, recorder.startCalled)
	assert.True(t, recorder.endCalled)
}

func TestCustomMeterProvider(t *testing.T) {
	recorder := &mockMetricsRecorder{}

	// Create a custom meter provider
	customProvider := noop.NewMeterProvider()

	router := chi.NewRouter()
	router.Use(metrics.Middleware(
		"test-server",
		metrics.WithMetricRecorders(recorder),
		metrics.WithMeterProvider(customProvider),
	))
	router.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	r := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	assert.True(t, recorder.registerCalled)
	assert.True(t, recorder.startCalled)
	assert.True(t, recorder.endCalled)

	// Verify that the custom meter provider was used
	meterFromRecorder := recorder.meter
	assert.NotNil(t, meterFromRecorder, "Meter should be set in recorder")
}

func TestPanicDuringRegistration(t *testing.T) {
	recorder := &mockMetricsRecorder{
		panicOnRegister: true,
	}

	assert.Panics(t, func() {
		metrics.Middleware(
			"test-server",
			metrics.WithMetricRecorders(recorder),
		)
	}, "Middleware should panic when recorder panics during registration")
}
