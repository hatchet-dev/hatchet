package nats

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	prommetrics "github.com/hatchet-dev/hatchet/pkg/integrations/metrics/prometheus"
)

var errBadSub = errors.New("nats: invalid subscription")

// fakeSub implements subscriptionStats without a live NATS server. A non-nil
// err mimics nats.go's ErrBadSubscription after Unsubscribe, which fails all
// three getters.
type fakeSub struct {
	dropped      int
	pendingMsgs  int
	pendingBytes int
	limitMsgs    int
	limitBytes   int
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

func (f *fakeSub) PendingLimits() (int, int, error) {
	if f.err != nil {
		return -1, -1, f.err
	}
	return f.limitMsgs, f.limitBytes, nil
}

func kindStats(t *testing.T, stats []prommetrics.NATSPubSubStats, kind msgqueue.TopicKind) prommetrics.NATSPubSubStats {
	t.Helper()

	for _, s := range stats {
		if s.TopicKind == string(kind) {
			return s
		}
	}

	t.Fatalf("no stats entry for topic kind %q", kind)
	return prommetrics.NATSPubSubStats{}
}

func TestSubRegistryDropsMonotonicAcrossRemoval(t *testing.T) {
	reg := newSubRegistry()
	kind := msgqueue.TopicKindTenantStream

	sub1 := &fakeSub{dropped: 3, limitMsgs: 500_000, limitBytes: 64 * 1024 * 1024}
	sub2 := &fakeSub{dropped: 4, limitMsgs: 500_000, limitBytes: 64 * 1024 * 1024}

	reg.add(sub1, kind)
	reg.add(sub2, kind)

	st := kindStats(t, reg.stats(), kind)
	assert.EqualValues(t, 7, st.DroppedTotal, "live subscriptions sum")
	assert.True(t, st.HasLiveSubs)

	// Removal folds the final Dropped() into the accumulator: the counter must
	// not go backwards when a subscription disappears.
	sub1.dropped = 5
	reg.remove(sub1)
	sub1.err = errBadSub // Unsubscribe would invalidate the getters

	st = kindStats(t, reg.stats(), kind)
	assert.EqualValues(t, 9, st.DroppedTotal, "folded final count of removed sub plus live sub")
	assert.True(t, st.HasLiveSubs)

	sub2.dropped = 6
	reg.remove(sub2)
	sub2.err = errBadSub

	st = kindStats(t, reg.stats(), kind)
	assert.EqualValues(t, 11, st.DroppedTotal, "accumulator only, no live subs")
	assert.False(t, st.HasLiveSubs, "no pending gauges without live subscriptions")

	// The kind keeps being exported after all its subscriptions are gone.
	st = kindStats(t, reg.stats(), kind)
	assert.EqualValues(t, 11, st.DroppedTotal)
}

func TestSubRegistryRemoveIsIdempotent(t *testing.T) {
	reg := newSubRegistry()
	kind := msgqueue.TopicKindSchedulerPartition

	sub := &fakeSub{dropped: 5}
	reg.add(sub, kind)

	reg.remove(sub)
	reg.remove(sub)

	st := kindStats(t, reg.stats(), kind)
	assert.EqualValues(t, 5, st.DroppedTotal, "double remove must not double-fold")
}

func TestSubRegistryRatchetHoldsWhenFinalCountIsLost(t *testing.T) {
	reg := newSubRegistry()
	kind := msgqueue.TopicKindTenantStream

	sub := &fakeSub{dropped: 10}
	reg.add(sub, kind)

	st := kindStats(t, reg.stats(), kind)
	require.EqualValues(t, 10, st.DroppedTotal)

	// The subscription becomes invalid while still registered (e.g. the
	// connection closed underneath it, or a scrape raced remove): its live
	// count is unreadable and nothing was folded, but the exported counter
	// must not drop below what a previous scrape reported.
	sub.err = errBadSub

	st = kindStats(t, reg.stats(), kind)
	assert.EqualValues(t, 10, st.DroppedTotal, "high-water mark holds the counter")

	reg.remove(sub)

	st = kindStats(t, reg.stats(), kind)
	assert.EqualValues(t, 10, st.DroppedTotal)
}

func TestSubRegistryPendingGaugesUseMax(t *testing.T) {
	reg := newSubRegistry()
	kind := msgqueue.TopicKindTenantStream

	reg.add(&fakeSub{pendingMsgs: 5, pendingBytes: 100, limitMsgs: 500_000, limitBytes: 64 * 1024 * 1024}, kind)
	reg.add(&fakeSub{pendingMsgs: 9, pendingBytes: 50, limitMsgs: 500_000, limitBytes: 64 * 1024 * 1024}, kind)

	st := kindStats(t, reg.stats(), kind)
	assert.True(t, st.HasLiveSubs)
	assert.Equal(t, 9, st.MaxPendingMsgs, "max message count across subscriptions")
	assert.Equal(t, 100, st.MaxPendingBytes, "max byte count across subscriptions")
	assert.Equal(t, 500_000, st.PendingLimitMsgs)
	assert.Equal(t, 64*1024*1024, st.PendingLimitBytes)
}

func TestSubRegistrySeparatesTopicKinds(t *testing.T) {
	reg := newSubRegistry()

	reg.add(&fakeSub{dropped: 1}, msgqueue.TopicKindTenantStream)
	reg.add(&fakeSub{dropped: 2}, msgqueue.TopicKindSchedulerPartition)

	stats := reg.stats()
	require.Len(t, stats, 2)

	assert.EqualValues(t, 1, kindStats(t, stats, msgqueue.TopicKindTenantStream).DroppedTotal)
	assert.EqualValues(t, 2, kindStats(t, stats, msgqueue.TopicKindSchedulerPartition).DroppedTotal)
}
