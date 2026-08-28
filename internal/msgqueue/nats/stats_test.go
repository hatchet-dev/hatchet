package nats

import (
	"errors"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	prommetrics "github.com/hatchet-dev/hatchet/pkg/integrations/metrics/prometheus"
)

var errBadSub = errors.New("nats: invalid subscription")

// fakeSub implements natsSub without a live NATS server. A non-nil err mimics
// nats.go's ErrBadSubscription after Unsubscribe.
type fakeSub struct {
	dropped      int
	pendingMsgs  int
	pendingBytes int
	err          error
}

func (f *fakeSub) Dropped() (int, error) {
	if f.err != nil {
		return -1, f.err
	}
	return f.dropped, nil
}

func (f *fakeSub) Pending() (int, int, error) {
	if f.err != nil {
		return -1, -1, f.err
	}
	return f.pendingMsgs, f.pendingBytes, nil
}

func metricValue(t *testing.T, m prometheus.Metric) float64 {
	t.Helper()

	pb := &dto.Metric{}
	require.NoError(t, m.Write(pb))

	if pb.Counter != nil {
		return pb.Counter.GetValue()
	}
	return pb.Gauge.GetValue()
}

func TestSubTrackerDropsMonotonicAndPendingMax(t *testing.T) {
	tr := newSubTracker()
	kind := string(msgqueue.TopicKindTenantStream)

	// the exported counter is process-global, so measure deltas from its
	// value at test start
	drops := prommetrics.NATSPubSubClientDrops.WithLabelValues(kind)
	base := metricValue(t, drops)
	droppedTotal := func() float64 { return metricValue(t, drops) - base }
	pendingMsgs := func() float64 {
		return metricValue(t, prommetrics.NATSPubSubPendingMsgs.WithLabelValues(kind))
	}

	sub1 := &fakeSub{dropped: 3, pendingMsgs: 5, pendingBytes: 100}
	sub2 := &fakeSub{dropped: 4, pendingMsgs: 9, pendingBytes: 50}
	tr.add(sub1, msgqueue.TopicKindTenantStream)
	tr.add(sub2, msgqueue.TopicKindTenantStream)

	tr.tick()
	assert.Equal(t, 7.0, droppedTotal(), "sum across live subscriptions")
	assert.Equal(t, 9.0, pendingMsgs(), "max across subscriptions")
	assert.Equal(t, 100.0, metricValue(t, prommetrics.NATSPubSubPendingBytes.WithLabelValues(kind)))

	// a subscription stuck in slow-consumer state keeps accumulating between ticks
	sub1.dropped = 5
	tr.tick()
	assert.Equal(t, 9.0, droppedTotal())

	// removal folds the delta since the last tick before Unsubscribe
	// invalidates the getters; a second remove must not double-count
	sub1.dropped = 6
	tr.remove(sub1)
	sub1.err = errBadSub
	tr.remove(sub1)
	assert.Equal(t, 10.0, droppedTotal())

	// unreadable subscriptions are skipped — the counter never decreases and
	// the pending gauges reset to 0 rather than holding a stale value
	sub2.err = errBadSub
	tr.tick()
	assert.Equal(t, 10.0, droppedTotal())
	assert.Equal(t, 0.0, pendingMsgs())
}
