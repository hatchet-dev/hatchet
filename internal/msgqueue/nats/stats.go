package nats

import (
	"maps"
	"sync"

	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	prommetrics "github.com/hatchet-dev/hatchet/pkg/integrations/metrics/prometheus"
)

// subscriptionStats is the subset of *natsgo.Subscription the registry reads.
// It exists so the drops-accumulator logic is testable without a live NATS
// server.
type subscriptionStats interface {
	Dropped() (int, error)
	Pending() (int, int, error)
	PendingLimits() (int, int, error)
}

// subRegistry tracks live subscriptions with their topic kind and feeds the
// Prometheus collector at scrape time. Removed subscriptions fold their final
// Dropped() count into a per-kind accumulator so the exported drops counter
// survives subscription churn.
//
// Engine pods hold one subscription per tenant stream, so add/remove stay
// O(1) map operations and stats never calls into nats.go while holding the
// lock.
type subRegistry struct {
	mu   sync.Mutex
	subs map[subscriptionStats]msgqueue.TopicKind

	// removedDrops accumulates the final Dropped() counts of removed
	// subscriptions per kind. A kind gains a (zero) entry on its first add, so
	// its drops counter keeps being exported even once all subscriptions of
	// the kind are gone (a series vanishing mid-scrape reads as a counter
	// reset to rate()).
	removedDrops map[msgqueue.TopicKind]uint64

	// reportedDrops is a per-kind high-water mark of exported totals. It
	// guards against one race: stats can snapshot removedDrops before a
	// concurrent remove folds a subscription's final count, then fail the
	// live Dropped() read because the subscription already unsubscribed —
	// without the ratchet that scrape would report a lower total than the
	// previous one.
	reportedDrops map[msgqueue.TopicKind]uint64
}

func newSubRegistry() *subRegistry {
	return &subRegistry{
		subs:          make(map[subscriptionStats]msgqueue.TopicKind),
		removedDrops:  make(map[msgqueue.TopicKind]uint64),
		reportedDrops: make(map[msgqueue.TopicKind]uint64),
	}
}

func (r *subRegistry) add(sub subscriptionStats, kind msgqueue.TopicKind) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.subs[sub] = kind

	if _, ok := r.removedDrops[kind]; !ok {
		r.removedDrops[kind] = 0
	}
}

// remove must run before the subscription is unsubscribed: Dropped() errors on
// a closed subscription and the final count would be lost. Safe to call more
// than once.
func (r *subRegistry) remove(sub subscriptionStats) {
	r.mu.Lock()
	defer r.mu.Unlock()

	kind, ok := r.subs[sub]
	if !ok {
		return
	}

	delete(r.subs, sub)

	if dropped, err := sub.Dropped(); err == nil && dropped > 0 {
		r.removedDrops[kind] += uint64(dropped)
	}
}

// stats snapshots the registry under the lock, then queries nats.go per
// subscription outside it. Subscriptions that error (unsubscribed between
// snapshot and query) are skipped; remove has folded or will fold their final
// drop counts.
func (r *subRegistry) stats() []prommetrics.NATSPubSubStats {
	r.mu.Lock()

	type liveSub struct {
		sub  subscriptionStats
		kind msgqueue.TopicKind
	}

	live := make([]liveSub, 0, len(r.subs))
	for sub, kind := range r.subs {
		live = append(live, liveSub{sub: sub, kind: kind})
	}

	base := maps.Clone(r.removedDrops)

	r.mu.Unlock()

	perKind := make(map[msgqueue.TopicKind]*prommetrics.NATSPubSubStats, len(base))

	kindStats := func(kind msgqueue.TopicKind) *prommetrics.NATSPubSubStats {
		st, ok := perKind[kind]
		if !ok {
			st = &prommetrics.NATSPubSubStats{
				TopicKind:    string(kind),
				DroppedTotal: base[kind],
			}
			perKind[kind] = st
		}
		return st
	}

	for kind := range base {
		kindStats(kind)
	}

	for _, ls := range live {
		st := kindStats(ls.kind)

		if dropped, err := ls.sub.Dropped(); err == nil && dropped > 0 {
			st.DroppedTotal += uint64(dropped)
		}

		pendingMsgs, pendingBytes, err := ls.sub.Pending()
		if err != nil {
			continue
		}

		limitMsgs, limitBytes, err := ls.sub.PendingLimits()
		if err != nil {
			continue
		}

		st.HasLiveSubs = true
		st.MaxPendingMsgs = max(st.MaxPendingMsgs, pendingMsgs)
		st.MaxPendingBytes = max(st.MaxPendingBytes, pendingBytes)
		st.PendingLimitMsgs = max(st.PendingLimitMsgs, limitMsgs)
		st.PendingLimitBytes = max(st.PendingLimitBytes, limitBytes)
	}

	r.mu.Lock()

	out := make([]prommetrics.NATSPubSubStats, 0, len(perKind))
	for kind, st := range perKind {
		if hw := r.reportedDrops[kind]; st.DroppedTotal < hw {
			st.DroppedTotal = hw
		} else {
			r.reportedDrops[kind] = st.DroppedTotal
		}
		out = append(out, *st)
	}

	r.mu.Unlock()

	return out
}
