package metric_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/riandyrn/otelchi/metric"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

func TestResponseSizeBytes(t *testing.T) {
	// setup environment
	responseMsg := "Hello, World!"

	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))

	baseCfg := metric.NewBaseConfig("test-server", metric.WithMeterProvider(provider))
	middleware := metric.NewResponseSizeBytes(baseCfg)

	router := chi.NewRouter()
	router.Use(middleware)
	router.Get("/test", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(responseMsg))
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	// read the recorded metrics
	var rm metricdata.ResourceMetrics
	err := reader.Collect(context.Background(), &rm)
	require.NoError(t, err)
	require.Len(t, rm.ScopeMetrics, 1)

	metrics := rm.ScopeMetrics[0].Metrics
	require.Len(t, metrics, 1)

	hist, ok := metrics[0].Data.(metricdata.Histogram[int64])
	require.True(t, ok)
	require.Len(t, hist.DataPoints, 1)

	dp := hist.DataPoints[0]
	assert.Equal(t, int64(len(responseMsg)), dp.Sum)
	assert.Equal(t, uint64(1), dp.Count)
}
