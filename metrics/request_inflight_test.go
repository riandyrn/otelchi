package metrics_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/riandyrn/otelchi/metrics"
	"github.com/stretchr/testify/require"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

func TestRequestInflight(t *testing.T) {
	// setup environment
	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))

	baseCfg := metrics.NewBaseConfig("test-server", metrics.WithMeterProvider(provider))
	middleware := metrics.NewRequestInFlight(baseCfg)

	router := chi.NewRouter()
	router.Use(middleware)
	router.Get("/test", func(w http.ResponseWriter, r *http.Request) {
		// the inflight request should be 1
		require.Equal(t, int64(1), getCountRequestInFlight(t, reader))
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	// the inflight request should be 0
	require.Equal(t, int64(0), getCountRequestInFlight(t, reader))

	// execute the request
	router.ServeHTTP(rec, req)

	// the inflight request should be 0
	require.Equal(t, int64(0), getCountRequestInFlight(t, reader))
}

func getCountRequestInFlight(t *testing.T, reader *sdkmetric.ManualReader) int64 {
	var rm metricdata.ResourceMetrics
	err := reader.Collect(context.Background(), &rm)
	require.NoError(t, err)
	if len(rm.ScopeMetrics) == 0 {
		return 0
	}

	metrics := rm.ScopeMetrics[0].Metrics
	if len(metrics) == 0 {
		return 0
	}

	dps, ok := metrics[0].Data.(metricdata.Sum[int64])
	require.True(t, ok)
	if len(dps.DataPoints) == 0 {
		return 0
	}

	dp := dps.DataPoints[0]
	return dp.Value
}
