package prometheus

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type PubSubHatchetMetric string

const (
	PubSubPublishDurationSeconds PubSubHatchetMetric = "hatchet_pubsub_publish_duration_seconds"
	PubSubTransitSeconds         PubSubHatchetMetric = "hatchet_pubsub_transit_seconds"
	PubSubNATSClientDropsTotal   PubSubHatchetMetric = "hatchet_pubsub_nats_client_drops_total"
)

var pubSubBuckets = []float64{0.01, 0.02, 0.05, 0.1, 0.5, 1, 2, 5, 15}

var (
	PubSubPublishDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    string(PubSubPublishDurationSeconds),
		Help:    "Time for the pub/sub backend's Pub call to return; this is publisher-side blocking cost, not broker delivery latency, and is not comparable across backends, which block at different depths before returning.",
		Buckets: pubSubBuckets,
	}, []string{"kind", "topic_kind", "result"})

	PubSubTransit = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    string(PubSubTransitSeconds),
		Help:    "Publish-to-delivery latency computed from the message's published_at stamp; subject to clock skew between publisher and subscriber pods; unstamped messages (older engines) are not observed.",
		Buckets: pubSubBuckets,
	}, []string{"kind", "topic_kind"})
)

// RegisterNATSClientDrops exposes dropped() as a counter labelled with the
// topic kind; the returned func unregisters it. Panics on double registration.
func RegisterNATSClientDrops(topicKind string, dropped func() float64) func() {
	c := prometheus.NewCounterFunc(prometheus.CounterOpts{
		Name:        string(PubSubNATSClientDropsTotal),
		Help:        "Messages dropped client-side by the nats.go pub/sub subscription on pending-limit violations, from Subscription.Dropped().",
		ConstLabels: prometheus.Labels{"topic_kind": topicKind},
	}, dropped)

	prometheus.MustRegister(c)

	return func() { prometheus.Unregister(c) }
}
