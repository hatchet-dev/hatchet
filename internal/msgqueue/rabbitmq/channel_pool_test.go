package rabbitmq

import (
	"context"
	"testing"

	"github.com/jackc/puddle/v2"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"

	"github.com/hatchet-dev/hatchet/pkg/logger"
)

func TestRedactURL(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "username and password",
			in:   "amqp://user:s3cret@rabbitmq-blue:5672/",
			want: "amqp://user:xxxxx@rabbitmq-blue:5672/",
		},
		{
			name: "username only",
			in:   "amqp://user@rabbitmq:5672/vhost",
			want: "amqp://user@rabbitmq:5672/vhost",
		},
		{
			name: "no credentials",
			in:   "amqp://rabbitmq:5672/",
			want: "amqp://rabbitmq:5672/",
		},
		{
			name: "unparseable",
			in:   "amqp://user:pass@rabbit:5672/\x7f%zz",
			want: "<unparseable url>",
		},
		{
			// A scheme-less string parses without a host, and Redacted would
			// return the credentials verbatim, so it must be masked entirely.
			name: "scheme-less with credentials",
			in:   "user:pass@rabbit:5672/vhost",
			want: "<unparseable url>",
		},
		{
			name: "empty",
			in:   "",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := redactURL(tt.in)

			if got != tt.want {
				t.Errorf("redactURL(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestChannelPoolMetricsReportAcquiredAndMax(t *testing.T) {
	ctx := context.Background()

	reader := metric.NewManualReader()
	provider := metric.NewMeterProvider(metric.WithReader(reader))
	prev := otel.GetMeterProvider()
	otel.SetMeterProvider(provider)
	t.Cleanup(func() {
		otel.SetMeterProvider(prev)
		_ = provider.Shutdown(ctx)
	})

	pool, err := puddle.NewPool(&puddle.Config[*amqp.Channel]{
		Constructor: func(context.Context) (*amqp.Channel, error) {
			return &amqp.Channel{}, nil
		},
		Destructor: func(*amqp.Channel) {},
		MaxSize:    4,
	})
	require.NoError(t, err)

	l := logger.NewDefaultLogger("test")
	p := &channelPool{Pool: pool, l: &l}
	t.Cleanup(p.Close)

	p.registerMetrics(channelPoolQueuePubSub, channelPoolRoleSub)

	res, err := pool.Acquire(ctx)
	require.NoError(t, err)
	t.Cleanup(res.Release)

	var rm metricdata.ResourceMetrics
	require.NoError(t, reader.Collect(ctx, &rm))

	require.Equal(t, int64(1), int64Gauge(t, rm, channelPoolAcquiredMetric, "pubsub", "sub"))
	require.Equal(t, int64(4), int64Gauge(t, rm, channelPoolMaxMetric, "pubsub", "sub"))
}

func int64Gauge(t *testing.T, rm metricdata.ResourceMetrics, name, queue, role string) int64 {
	t.Helper()

	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name != name {
				continue
			}

			gauge, ok := m.Data.(metricdata.Gauge[int64])
			require.True(t, ok, "metric %s is not an int64 gauge", name)

			for _, dp := range gauge.DataPoints {
				q, _ := dp.Attributes.Value("queue")
				r, _ := dp.Attributes.Value("pool")
				if q.AsString() == queue && r.AsString() == role {
					return dp.Value
				}
			}
		}
	}

	t.Fatalf("no datapoint for %s queue=%s pool=%s", name, queue, role)
	return 0
}
