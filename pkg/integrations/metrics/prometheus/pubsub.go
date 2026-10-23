package prometheus

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type PubSubHatchetMetric string

const (
	PubSubPublishDurationSeconds                PubSubHatchetMetric = "hatchet_pubsub_publish_duration_seconds"
	PubSubTransitSeconds                        PubSubHatchetMetric = "hatchet_pubsub_transit_seconds"
	PubSubNATSSchedulerPartitionDropsTotal      PubSubHatchetMetric = "hatchet_pubsub_nats_scheduler_partition_drops_total"
	PubSubStaleSkippedTotal                     PubSubHatchetMetric = "hatchet_pubsub_stale_skipped_total"
	PubSubHandlersInFlightCount                 PubSubHatchetMetric = "hatchet_pubsub_handlers_in_flight"
	PubSubHandlerPoolFullTotal                  PubSubHatchetMetric = "hatchet_pubsub_handler_pool_full_total"
	PubSubHandlerSlotWaitSeconds                PubSubHatchetMetric = "hatchet_pubsub_handler_slot_wait_seconds"
	PubSubNATSSchedulerPartitionPendingMessages PubSubHatchetMetric = "hatchet_pubsub_nats_scheduler_partition_pending_messages"
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

	PubSubStaleSkipped = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: string(PubSubStaleSkippedTotal),
		Help: "Delivered messages skipped without running the handler because their published_at stamp was older than the topic kind's max age; subject to clock skew between publisher and subscriber pods.",
	}, []string{"kind", "topic_kind"})

	PubSubHandlersInFlight = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: string(PubSubHandlersInFlightCount),
		Help: "Subscriber handler calls currently running, summed over the process's subscriptions of a topic kind.",
	}, []string{"kind", "topic_kind"})

	PubSubHandlerPoolFull = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: string(PubSubHandlerPoolFullTotal),
		Help: "Deliveries that found their subscription's handler concurrency limit reached and had to wait for a running handler to return.",
	}, []string{"kind", "topic_kind"})

	PubSubHandlerSlotWait = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    string(PubSubHandlerSlotWaitSeconds),
		Help:    "Time a delivery waited for a free handler slot in its subscription; zero when a slot was free.",
		Buckets: pubSubBuckets,
	}, []string{"kind", "topic_kind"})
)

// RegisterNATSSchedulerPartitionDrops exposes dropped() as a counter; the
// returned func unregisters it. Panics on double registration.
func RegisterNATSSchedulerPartitionDrops(dropped func() float64) func() {
	c := prometheus.NewCounterFunc(prometheus.CounterOpts{
		Name: string(PubSubNATSSchedulerPartitionDropsTotal),
		Help: "Messages dropped client-side by nats.go on pending-limit violations, from Subscription.Dropped(); covers the scheduler-partition subscription only.",
	}, dropped)

	prometheus.MustRegister(c)

	return func() { prometheus.Unregister(c) }
}

// RegisterNATSSchedulerPartitionPending exposes pending() as a gauge; the
// returned func unregisters it. Panics on double registration.
func RegisterNATSSchedulerPartitionPending(pending func() float64) func() {
	g := prometheus.NewGaugeFunc(prometheus.GaugeOpts{
		Name: string(PubSubNATSSchedulerPartitionPendingMessages),
		Help: "Messages buffered client-side by nats.go and not yet handed to the subscriber, from Subscription.Pending(); covers the scheduler-partition subscription only.",
	}, pending)

	prometheus.MustRegister(g)

	return func() { prometheus.Unregister(g) }
}
