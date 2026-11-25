package prometheus

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	InternalEventOutboxBatches = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "hatchet_internal_event_outbox_batches_total",
		Help: "Internal event outbox batches this engine processed, by result (ok or error)",
	}, []string{"result"})

	// The reason label is one of: failed (the handler returned an error), backoff (a failed message whose
	// next attempt isn't due yet), undecodable. A steady failed rate means events that never succeed.
	InternalEventOutboxRequeued = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "hatchet_internal_event_outbox_requeued_total",
		Help: "Internal event outbox messages moved to their shard's retry topic instead of processed, by reason",
	}, []string{"reason"})

	// Set per engine from the last batch it took from a retry topic, which with more than a batch of
	// retries is not the whole topic, so dashboards should take max by topic over a window. It reads 0
	// once a topic's retries are gone; a growing value means events that keep failing. One label value
	// per retry topic (8).
	InternalEventOutboxRetryOldest = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "hatchet_internal_event_outbox_retry_oldest_seconds",
		Help: "Age of the oldest failing internal event in the last batch taken from a retry topic",
	}, []string{"topic"})

	// Measured from the first outbox insert, so it includes the time the completion transaction stayed open.
	InternalEventOutboxLag = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "hatchet_internal_event_outbox_lag_seconds",
		Help:    "Time between first writing internal events to the outbox and processing them, retries included",
		Buckets: []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10, 30},
	})

	// The result label is one of: sent, dropped (test-only fault injection), error.
	DispatcherPubSubDeliveryMessages = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "hatchet_dispatcher_pubsub_delivery_messages_total",
		Help: "Messages for a dispatcher sent over pub/sub instead of its durable queue, by message id and result",
	}, []string{"msg_id", "result"})

	DispatcherPulledAssignments = promauto.NewCounter(prometheus.CounterOpts{
		Name: "hatchet_dispatcher_pulled_assignments_total",
		Help: "Task assignments a dispatcher found in the database without having received them over pub/sub",
	})
)
