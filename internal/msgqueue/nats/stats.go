package nats

import (
	"sync"
	"time"

	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	prommetrics "github.com/hatchet-dev/hatchet/pkg/integrations/metrics/prometheus"
)

const statsInterval = 15 * time.Second

// natsSub is the subset of *natsgo.Subscription the tracker reads, split out
// so drop accounting is testable without a live NATS server.
type natsSub interface {
	Dropped() (int, error)
	Pending() (int, int, error)
}

// subTracker feeds the hatchet_pubsub_nats_* client metrics. Dropped() is the
// only reliable drop signal — the async ErrorHandler fires just on the
// transition into slow-consumer state and stays silent while stuck in it — so
// the tracker adds per-subscription Dropped() deltas to the exported counter
// on every tick and once more on remove, keeping it monotonic and eventually
// exact.
type subTracker struct {
	mu    sync.Mutex
	subs  map[natsSub]*subState
	kinds map[msgqueue.TopicKind]struct{}
}

type subState struct {
	kind        msgqueue.TopicKind
	lastDropped int
}

func newSubTracker() *subTracker {
	return &subTracker{
		subs:  make(map[natsSub]*subState),
		kinds: make(map[msgqueue.TopicKind]struct{}),
	}
}

// run refreshes the exported metrics every statsInterval until stop is closed.
func (t *subTracker) run(stop <-chan struct{}) {
	ticker := time.NewTicker(statsInterval)
	defer ticker.Stop()

	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			t.tick()
		}
	}
}

func (t *subTracker) add(sub natsSub, kind msgqueue.TopicKind) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.subs[sub] = &subState{kind: kind}
	t.kinds[kind] = struct{}{}

	// materialize the counter series at 0 so rate() sees the first drop as an
	// increase rather than the series appearing
	prommetrics.NATSPubSubClientDrops.WithLabelValues(string(kind))
}

// remove must run before Unsubscribe, which invalidates Dropped() and would
// lose the drops since the last tick. Idempotent.
func (t *subTracker) remove(sub natsSub) {
	t.mu.Lock()
	st, ok := t.subs[sub]
	delete(t.subs, sub)
	t.mu.Unlock()

	if ok {
		t.accountDrops(sub, st)
	}
}

// accountDrops adds the subscription's drop delta since the last read to the
// exported counter. Deltas are computed under the lock, so tick racing remove
// on the same subscription never double-counts.
func (t *subTracker) accountDrops(sub natsSub, st *subState) {
	dropped, err := sub.Dropped()
	if err != nil {
		return
	}

	t.mu.Lock()
	delta := dropped - st.lastDropped
	if delta > 0 {
		st.lastDropped = dropped
	}
	t.mu.Unlock()

	if delta > 0 {
		prommetrics.NATSPubSubClientDrops.WithLabelValues(string(st.kind)).Add(float64(delta))
	}
}

// tick accounts drop deltas for live subscriptions and sets the pending
// gauges to the max across live subscriptions per topic kind (max surfaces
// one hot subscription; 0 once a kind has none left, never a stale value).
func (t *subTracker) tick() {
	type liveSub struct {
		sub natsSub
		st  *subState
	}

	t.mu.Lock()
	live := make([]liveSub, 0, len(t.subs))
	for sub, st := range t.subs {
		live = append(live, liveSub{sub: sub, st: st})
	}
	maxMsgs := make(map[msgqueue.TopicKind]int, len(t.kinds))
	maxBytes := make(map[msgqueue.TopicKind]int, len(t.kinds))
	for kind := range t.kinds {
		maxMsgs[kind], maxBytes[kind] = 0, 0
	}
	t.mu.Unlock()

	for _, ls := range live {
		t.accountDrops(ls.sub, ls.st)

		if msgs, bytes, err := ls.sub.Pending(); err == nil {
			maxMsgs[ls.st.kind] = max(maxMsgs[ls.st.kind], msgs)
			maxBytes[ls.st.kind] = max(maxBytes[ls.st.kind], bytes)
		}
	}

	for kind := range maxMsgs {
		prommetrics.NATSPubSubPendingMsgs.WithLabelValues(string(kind)).Set(float64(maxMsgs[kind]))
		prommetrics.NATSPubSubPendingBytes.WithLabelValues(string(kind)).Set(float64(maxBytes[kind]))
	}
}
