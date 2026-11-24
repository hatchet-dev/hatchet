package prometheus

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// DirectTaskWriteHatchetMetric names the metrics reported by the dispatcher's direct task writer, which
// applies worker task events in-process instead of sending them over the durable task queue.
type DirectTaskWriteHatchetMetric string

const (
	DirectTaskWritesTotal          DirectTaskWriteHatchetMetric = "hatchet_direct_task_writes_total"
	DirectTaskWriteDurationSeconds DirectTaskWriteHatchetMetric = "hatchet_direct_task_write_duration_seconds"
)

// The result label is one of: direct (applied in-process), direct_alone (applied in-process on its own
// after the batch it was in failed), and fallback_no_slot or fallback_error (sent over the durable task
// queue instead).
var (
	DirectTaskWrites = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: string(DirectTaskWritesTotal),
		Help: "Task-processing messages the dispatcher tried to apply in-process, by message type and result",
	}, []string{"msg_type", "result"})

	// Covers the wait for the batch flush as well as the handler, since that is the latency the worker's RPC sees.
	DirectTaskWriteDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    string(DirectTaskWriteDurationSeconds),
		Help:    "Time to apply a task-processing message in-process, including the wait for its batch to flush",
		Buckets: []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
	}, []string{"msg_type", "result"})
)
