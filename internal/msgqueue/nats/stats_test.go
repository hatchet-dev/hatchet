package nats

import (
	"errors"
	"testing"

	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	prommetrics "github.com/hatchet-dev/hatchet/pkg/integrations/metrics/prometheus"
)

// fakeSub implements natsSub without a live NATS server. A non-nil err mimics
// nats.go's ErrBadSubscription after Unsubscribe.
type fakeSub struct {
	dropped int
	err     error
}

func (f *fakeSub) Dropped() (int, error) {
	if f.err != nil {
		return -1, f.err
	}
	return f.dropped, nil
}

func TestSubTrackerDropsMonotonic(t *testing.T) {
	tr := newSubTracker()
	kind := msgqueue.TopicKindTenantStream

	// the exported counter is process-global, so measure deltas from its
	// value at test start
	counter := prommetrics.NATSPubSubClientDrops.WithLabelValues(string(kind))
	counterValue := func() float64 {
		pb := &dto.Metric{}
		require.NoError(t, counter.Write(pb))
		return pb.Counter.GetValue()
	}
	base := counterValue()
	total := func() float64 { return counterValue() - base }

	sub := &fakeSub{dropped: 3}
	tr.add(sub, kind)

	// a handler fire adds the delta; a fire without new drops adds nothing
	tr.accountDrops(sub)
	assert.Equal(t, 3.0, total())
	tr.accountDrops(sub)
	assert.Equal(t, 3.0, total())

	// drops accumulated silently between fires are captured by the next
	// fire's delta
	sub.dropped = 8
	tr.accountDrops(sub)
	assert.Equal(t, 8.0, total())

	// removal folds the final delta before Unsubscribe invalidates Dropped();
	// a second remove must not double-count
	sub.dropped = 10
	tr.remove(sub)
	sub.err = errors.New("nats: invalid subscription")
	tr.remove(sub)
	assert.Equal(t, 10.0, total())

	// untracked or unreadable subscriptions never move the counter
	tr.accountDrops(sub)
	tr.accountDrops(&fakeSub{dropped: 100})
	assert.Equal(t, 10.0, total())
}
