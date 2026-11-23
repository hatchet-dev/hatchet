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

	// Measured from the outbox insert, so it includes the time the completion transaction stayed open.
	InternalEventOutboxLag = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "hatchet_internal_event_outbox_lag_seconds",
		Help:    "Time between writing internal events to the outbox and processing them",
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
