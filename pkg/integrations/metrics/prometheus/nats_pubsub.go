package prometheus

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	PubSubNATSClientDropsTotal  PubSubHatchetMetric = "hatchet_pubsub_nats_client_drops_total"
	PubSubNATSPendingMsgs       PubSubHatchetMetric = "hatchet_pubsub_nats_pending_msgs"
	PubSubNATSPendingBytes      PubSubHatchetMetric = "hatchet_pubsub_nats_pending_bytes"
	PubSubNATSPendingLimitMsgs  PubSubHatchetMetric = "hatchet_pubsub_nats_pending_limit_msgs"
	PubSubNATSPendingLimitBytes PubSubHatchetMetric = "hatchet_pubsub_nats_pending_limit_bytes"
)

// NATSPubSubStats is a point-in-time snapshot of the NATS client-side
// subscription state for one topic kind, produced by the nats pub/sub
// backend's subscription registry and exported by natsPubSubCollector.
//
// There is deliberately no subject label anywhere in these metrics: subjects
// embed per-tenant UUIDs and would explode cardinality. Per-shard granularity
// comes from scrape-level namespace attributes.
type NATSPubSubStats struct {
	TopicKind string

	// DroppedTotal is the cumulative count of messages dropped client-side by
	// nats.go (pending-limit violations), summed over live subscriptions of
	// this kind plus the final counts of subscriptions that have since been
	// removed. It never decreases.
	DroppedTotal uint64

	// HasLiveSubs reports whether at least one live subscription of this kind
	// was successfully queried; the pending gauges below are only meaningful
	// (and only exported) when true.
	HasLiveSubs bool

	// MaxPending* are the maximum Pending() values across live subscriptions
	// of this kind; max (rather than sum or avg) surfaces one hot subscription.
	MaxPendingMsgs  int
	MaxPendingBytes int

	// PendingLimit* are the configured client-side limits (max across live
	// subscriptions; in practice all subscriptions share the same limits).
	PendingLimitMsgs  int
	PendingLimitBytes int
}

var (
	natsClientDropsDesc = prometheus.NewDesc(
		string(PubSubNATSClientDropsTotal),
		"Messages dropped client-side by nats.go pub/sub subscriptions on pending-limit violations, from Subscription.Dropped(); cumulative across removed subscriptions. This is the source of truth for drops: the async error handler fires only on the transition into slow-consumer state and under-counts.",
		[]string{"topic_kind"}, nil,
	)
	natsPendingMsgsDesc = prometheus.NewDesc(
		string(PubSubNATSPendingMsgs),
		"Messages buffered client-side, max across live NATS pub/sub subscriptions of the topic kind (Subscription.Pending()).",
		[]string{"topic_kind"}, nil,
	)
	natsPendingBytesDesc = prometheus.NewDesc(
		string(PubSubNATSPendingBytes),
		"Bytes buffered client-side, max across live NATS pub/sub subscriptions of the topic kind (Subscription.Pending()).",
		[]string{"topic_kind"}, nil,
	)
	natsPendingLimitMsgsDesc = prometheus.NewDesc(
		string(PubSubNATSPendingLimitMsgs),
		"Configured client-side pending message limit for NATS pub/sub subscriptions of the topic kind; divide hatchet_pubsub_nats_pending_msgs by this for buffer occupancy.",
		[]string{"topic_kind"}, nil,
	)
	natsPendingLimitBytesDesc = prometheus.NewDesc(
		string(PubSubNATSPendingLimitBytes),
		"Configured client-side pending bytes limit for NATS pub/sub subscriptions of the topic kind; divide hatchet_pubsub_nats_pending_bytes by this for buffer occupancy.",
		[]string{"topic_kind"}, nil,
	)
)

// natsPubSubCollector exports NATS client-side subscription stats at scrape
// time (no ticker): every Gather calls the registered providers, which read
// live counters from nats.go. Providers are registered by NewPubSub and
// removed by its cleanup function.
type natsPubSubCollector struct {
	mu        sync.Mutex
	nextID    uint64
	providers map[uint64]func() []NATSPubSubStats
}

var natsCollector = func() *natsPubSubCollector {
	c := &natsPubSubCollector{providers: make(map[uint64]func() []NATSPubSubStats)}
	prometheus.MustRegister(c)
	return c
}()

// RegisterNATSPubSubStatsProvider registers a stats provider with the NATS
// pub/sub collector and returns a function that removes it. Multiple providers
// (multiple PubSub instances in one process, as in tests) are merged per topic
// kind so Gather never sees duplicate label sets.
func RegisterNATSPubSubStatsProvider(provider func() []NATSPubSubStats) (unregister func()) {
	c := natsCollector

	c.mu.Lock()
	id := c.nextID
	c.nextID++
	c.providers[id] = provider
	c.mu.Unlock()

	return func() {
		c.mu.Lock()
		delete(c.providers, id)
		c.mu.Unlock()
	}
}

func (c *natsPubSubCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- natsClientDropsDesc
	ch <- natsPendingMsgsDesc
	ch <- natsPendingBytesDesc
	ch <- natsPendingLimitMsgsDesc
	ch <- natsPendingLimitBytesDesc
}

func (c *natsPubSubCollector) Collect(ch chan<- prometheus.Metric) {
	c.mu.Lock()
	providers := make([]func() []NATSPubSubStats, 0, len(c.providers))
	for _, p := range c.providers {
		providers = append(providers, p)
	}
	c.mu.Unlock()

	merged := make(map[string]*NATSPubSubStats)

	for _, provider := range providers {
		for _, s := range provider() {
			m, ok := merged[s.TopicKind]
			if !ok {
				cp := s
				merged[s.TopicKind] = &cp
				continue
			}

			m.DroppedTotal += s.DroppedTotal
			if s.HasLiveSubs {
				m.HasLiveSubs = true
				m.MaxPendingMsgs = max(m.MaxPendingMsgs, s.MaxPendingMsgs)
				m.MaxPendingBytes = max(m.MaxPendingBytes, s.MaxPendingBytes)
				m.PendingLimitMsgs = max(m.PendingLimitMsgs, s.PendingLimitMsgs)
				m.PendingLimitBytes = max(m.PendingLimitBytes, s.PendingLimitBytes)
			}
		}
	}

	for _, s := range merged {
		ch <- prometheus.MustNewConstMetric(natsClientDropsDesc, prometheus.CounterValue, float64(s.DroppedTotal), s.TopicKind)

		if !s.HasLiveSubs {
			continue
		}

		ch <- prometheus.MustNewConstMetric(natsPendingMsgsDesc, prometheus.GaugeValue, float64(s.MaxPendingMsgs), s.TopicKind)
		ch <- prometheus.MustNewConstMetric(natsPendingBytesDesc, prometheus.GaugeValue, float64(s.MaxPendingBytes), s.TopicKind)
		ch <- prometheus.MustNewConstMetric(natsPendingLimitMsgsDesc, prometheus.GaugeValue, float64(s.PendingLimitMsgs), s.TopicKind)
		ch <- prometheus.MustNewConstMetric(natsPendingLimitBytesDesc, prometheus.GaugeValue, float64(s.PendingLimitBytes), s.TopicKind)
	}
}
