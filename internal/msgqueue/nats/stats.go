package nats

import (
	"sync"

	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	prommetrics "github.com/hatchet-dev/hatchet/pkg/integrations/metrics/prometheus"
)

// natsSub is the subset of *natsgo.Subscription the tracker reads, split out
// so drop accounting is testable without a live NATS server.
type natsSub interface {
	Dropped() (int, error)
}

// subTracker feeds hatchet_pubsub_nats_client_drops_total. Dropped() is the
// only reliable drop signal — the async ErrorHandler fires just on the
// transition into slow-consumer state and stays silent while stuck in it —
// so accountDrops adds each subscription's Dropped() delta on every handler
// fire and once more on remove. Accepted gap: a subscription that enters
// slow-consumer state and never drains again stops updating between fires.
type subTracker struct {
	mu   sync.Mutex
	subs map[natsSub]*subState
}

type subState struct {
	kind        msgqueue.TopicKind
	lastDropped int
}

func newSubTracker() *subTracker {
	return &subTracker{subs: make(map[natsSub]*subState)}
}

func (t *subTracker) add(sub natsSub, kind msgqueue.TopicKind) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.subs[sub] = &subState{kind: kind}

	// materialize the counter series at 0 so rate() sees the first drop as an
	// increase rather than the series appearing
	prommetrics.NATSPubSubClientDrops.WithLabelValues(string(kind))
}

// remove accounts the final delta and must run before Unsubscribe, which
// invalidates Dropped(). Idempotent.
func (t *subTracker) remove(sub natsSub) {
	t.accountDrops(sub)

	t.mu.Lock()
	delete(t.subs, sub)
	t.mu.Unlock()
}

// accountDrops adds the subscription's drop delta since the last read to the
// exported counter. Deltas are computed under the lock and only positive
// deltas are added, so concurrent calls never double-count and the counter is
// monotonic.
func (t *subTracker) accountDrops(sub natsSub) {
	dropped, err := sub.Dropped()
	if err != nil {
		return
	}

	t.mu.Lock()
	st, ok := t.subs[sub]
	delta := 0
	if ok {
		delta = dropped - st.lastDropped
		if delta > 0 {
			st.lastDropped = dropped
		}
	}
	t.mu.Unlock()

	if delta > 0 {
		prommetrics.NATSPubSubClientDrops.WithLabelValues(string(st.kind)).Add(float64(delta))
	}
}
