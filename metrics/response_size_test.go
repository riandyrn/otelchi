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

func TestResponseSize(t *testing.T) {
	attrs := []attribute.KeyValue{
		attribute.String("test_key", "test_value"),
	}

	ctx := context.Background()
	testCases := []struct {
		name         string
		writtenBytes int
		opts         metrics.MetricOpts
	}{
		{
			name:         "zero bytes",
			writtenBytes: 0,
			opts: metrics.MetricOpts{
				Measurement: metric.WithAttributes(attrs...),
				ResponseData: metrics.ResponseData{
					WrittenBytes: 0,
				},
			},
		},
		{
			name:         "small response",
			writtenBytes: 100,
			opts: metrics.MetricOpts{
				Measurement: metric.WithAttributes(attrs...),
				ResponseData: metrics.ResponseData{
					WrittenBytes: 100,
				},
			},
		},
		{
			name:         "large response",
			writtenBytes: 1024 * 1024, // 1MB
			opts: metrics.MetricOpts{
				Measurement: metric.WithAttributes(attrs...),
				ResponseData: metrics.ResponseData{
					WrittenBytes: 1024 * 1024,
				},
			},
		},
	}

	// Record metrics for different response sizes
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			reader := sdkmetric.NewManualReader()
			provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))

			recorder := metrics.NewResponseSizeBytes()
			recorder.RegisterMetric(context.Background(), metrics.RegisterMetricConfig{
				Meter: provider.Meter("test"),
			})

			recorder.StartMetric(ctx, tc.opts) // Should do nothing
			recorder.EndMetric(ctx, tc.opts)

			var rm metricdata.ResourceMetrics
			err := reader.Collect(context.Background(), &rm)
			require.NoError(t, err)
			require.Len(t, rm.ScopeMetrics, 1)

			metrics := rm.ScopeMetrics[0].Metrics
			require.Len(t, metrics, 1)

			metric := metrics[0]
			assert.Equal(t, "response_size_bytes", metric.Name)

			hist, ok := metric.Data.(metricdata.Histogram[int64])
			require.True(t, ok)
			require.Len(t, hist.DataPoints, 1)

			dp := hist.DataPoints[0]
			assert.Equal(t, uint64(tc.writtenBytes), uint64(dp.Sum))
			assert.Equal(t, uint64(1), dp.Count)
			assert.Equal(t, attribute.NewSet(attrs...), dp.Attributes)
		})
	}
}
