package metric_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/riandyrn/otelchi/metric"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

func TestServerRequestDuration(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	baseCfg := metric.NewBaseConfig(
		"test-server",
		metric.WithMeterProvider(provider),
		metric.WithAttributesFunc(testAttributes),
	)

	router := chi.NewRouter()
	router.Use(metric.NewServerRequestDuration(baseCfg))
	router.Get("/test", func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(10 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	})

	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/test", nil))

	m := findMetric(t, collectMetrics(t, reader), "http.server.request.duration")
	assert.Equal(t, "s", m.Unit)
	assert.Equal(t, "Duration of HTTP server requests.", m.Description)

	histogram, ok := m.Data.(metricdata.Histogram[float64])
	require.True(t, ok)
	require.Len(t, histogram.DataPoints, 1)

	dp := histogram.DataPoints[0]
	assert.Greater(t, dp.Sum, 0.0)
	assert.Equal(t, uint64(1), dp.Count)
	assertHasAttribute(t, dp.Attributes, attribute.String("test.attr", "value"))
}

func TestServerRequestDuration_RecordsStatusCode(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	baseCfg := metric.NewBaseConfig("test-server", metric.WithMeterProvider(provider))

	router := chi.NewRouter()
	router.Use(metric.NewServerRequestDuration(baseCfg))
	router.Get("/created", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	})

	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/created", nil))

	m := findMetric(t, collectMetrics(t, reader), "http.server.request.duration")
	histogram, ok := m.Data.(metricdata.Histogram[float64])
	require.True(t, ok)
	require.Len(t, histogram.DataPoints, 1)

	v, ok := histogram.DataPoints[0].Attributes.Value(attribute.Key("http.response.status_code"))
	require.True(t, ok, "http.response.status_code attribute missing")
	assert.Equal(t, int64(201), v.AsInt64())
}

func TestServerRequestDuration_DefaultsStatusCodeTo200(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	baseCfg := metric.NewBaseConfig("test-server", metric.WithMeterProvider(provider))

	router := chi.NewRouter()
	router.Use(metric.NewServerRequestDuration(baseCfg))
	router.Get("/ok", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok")) // implicit 200, no WriteHeader
	})

	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/ok", nil))

	m := findMetric(t, collectMetrics(t, reader), "http.server.request.duration")
	histogram, ok := m.Data.(metricdata.Histogram[float64])
	require.True(t, ok)
	require.Len(t, histogram.DataPoints, 1)

	v, ok := histogram.DataPoints[0].Attributes.Value(attribute.Key("http.response.status_code"))
	require.True(t, ok)
	assert.Equal(t, int64(200), v.AsInt64())
}
