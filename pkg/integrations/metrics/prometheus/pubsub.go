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
	PubSubNATSPendingMsgs        PubSubHatchetMetric = "hatchet_pubsub_nats_pending_msgs"
	PubSubNATSPendingBytes       PubSubHatchetMetric = "hatchet_pubsub_nats_pending_bytes"
)

var pubSubBuckets = []float64{0.0001, 0.00025, 0.0005, 0.001, 0.0025, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5}

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

	// The NATS client metrics carry no subject label: subjects embed
	// per-tenant UUIDs and would explode cardinality.
	NATSPubSubClientDrops = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: string(PubSubNATSClientDropsTotal),
		Help: "Messages dropped client-side by nats.go pub/sub subscriptions on pending-limit violations, from Subscription.Dropped(); cumulative across subscription churn.",
	}, []string{"topic_kind"})

	NATSPubSubPendingMsgs = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: string(PubSubNATSPendingMsgs),
		Help: "Messages buffered client-side, max across live NATS pub/sub subscriptions of the topic kind (Subscription.Pending()).",
	}, []string{"topic_kind"})

	NATSPubSubPendingBytes = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: string(PubSubNATSPendingBytes),
		Help: "Bytes buffered client-side, max across live NATS pub/sub subscriptions of the topic kind (Subscription.Pending()).",
	}, []string{"topic_kind"})
)
