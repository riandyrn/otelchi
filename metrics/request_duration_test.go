package metrics_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"

	"github.com/riandyrn/otelchi/metrics"
)

func TestRequestDuration(t *testing.T) {
	testCases := []struct {
		name     string
		delay    time.Duration
		attrs    []attribute.KeyValue
		expected struct {
			minLatency int64
			count      uint64
		}
	}{
		{
			name:  "fast request",
			delay: 10 * time.Millisecond,
			attrs: nil,
			expected: struct {
				minLatency int64
				count      uint64
			}{
				minLatency: 10,
				count:      1,
			},
		},
		{
			name:  "slow request",
			delay: 100 * time.Millisecond,
			attrs: nil,
			expected: struct {
				minLatency int64
				count      uint64
			}{
				minLatency: 100,
				count:      1,
			},
		},
		{
			name:  "request with attributes",
			delay: 10 * time.Millisecond,
			attrs: []attribute.KeyValue{
				attribute.String("test_key", "test_value"),
			},
			expected: struct {
				minLatency int64
				count      uint64
			}{
				minLatency: 10,
				count:      1,
			},
		},
		{
			name:  "zero latency request",
			delay: 0,
			attrs: nil,
			expected: struct {
				minLatency int64
				count      uint64
			}{
				minLatency: 0,
				count:      1,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			reader := sdkmetric.NewManualReader()
			provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))

			recorder := metrics.NewRequestDurationMs()
			recorder.RegisterMetric(context.Background(), metrics.RegisterMetricConfig{
				Meter: provider.Meter("test"),
			})

			ctx := context.Background()
			opts := metrics.MetricOpts{
				Measurement: metric.WithAttributes(tc.attrs...),
			}

			recorder.StartMetric(ctx, opts)
			if tc.delay > 0 {
				time.Sleep(tc.delay)
			}
			recorder.EndMetric(ctx, opts)

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
			assert.GreaterOrEqual(t, dp.Sum, tc.expected.minLatency)
			assert.Equal(t, tc.expected.count, dp.Count)

			if tc.attrs != nil {
				assert.Equal(t, attribute.NewSet(tc.attrs...), dp.Attributes)
			}
		})
	}
}
