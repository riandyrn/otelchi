package metrics_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"

	"github.com/riandyrn/otelchi/metrics"
)

func TestRequestInFlight(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))

	recorder := metrics.NewRequestInFlight()
	recorder.RegisterMetric(context.Background(), metrics.RegisterMetricConfig{
		Meter: provider.Meter("test"),
	})

	attrs := []attribute.KeyValue{
		attribute.String("test_key", "test_value"),
	}

	ctx := context.Background()
	opts := metrics.MetricOpts{
		Measurement: metric.WithAttributes(attrs...),
	}

	// Start 3 requests
	recorder.StartMetric(ctx, opts)
	recorder.StartMetric(ctx, opts)
	recorder.StartMetric(ctx, opts)

	// Collect and verify metrics - should have 3 requests in flight
	var rm metricdata.ResourceMetrics
	err := reader.Collect(context.Background(), &rm)
	require.NoError(t, err)
	require.Len(t, rm.ScopeMetrics, 1)

	metrics := rm.ScopeMetrics[0].Metrics
	require.Len(t, metrics, 1)

	metric := metrics[0]
	assert.Equal(t, "requests_inflight", metric.Name)

	sum, ok := metric.Data.(metricdata.Sum[int64])
	require.True(t, ok)
	require.Len(t, sum.DataPoints, 1)

	// Should be 3 requests in flight
	assert.Equal(t, int64(3), sum.DataPoints[0].Value)

	// End 2 requests
	recorder.EndMetric(ctx, opts)
	recorder.EndMetric(ctx, opts)

	// Collect and verify metrics - should have 1 request in flight
	err = reader.Collect(context.Background(), &rm)
	require.NoError(t, err)

	metrics = rm.ScopeMetrics[0].Metrics
	sum = metrics[0].Data.(metricdata.Sum[int64])
	assert.Equal(t, int64(1), sum.DataPoints[0].Value)

	// End last request
	recorder.EndMetric(ctx, opts)

	// Collect and verify metrics - should have 0 requests in flight
	err = reader.Collect(context.Background(), &rm)
	require.NoError(t, err)

	metrics = rm.ScopeMetrics[0].Metrics
	sum = metrics[0].Data.(metricdata.Sum[int64])
	assert.Equal(t, int64(0), sum.DataPoints[0].Value)
	assert.Equal(t, attribute.NewSet(attrs...), sum.DataPoints[0].Attributes)
}
