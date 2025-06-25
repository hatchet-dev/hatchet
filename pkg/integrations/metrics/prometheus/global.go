package prometheus

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type GlobalHatchetMetric string

const (
	QueueInvocationsTotal GlobalHatchetMetric = "hatchet_queue_invocations_total"
	CreatedTasksTotal     GlobalHatchetMetric = "hatchet_created_tasks_total"
	RetriedTasksTotal     GlobalHatchetMetric = "hatchet_retried_tasks_total"
	SucceededTasksTotal   GlobalHatchetMetric = "hatchet_succeeded_tasks_total"
	FailedTasksTotal      GlobalHatchetMetric = "hatchet_failed_tasks_total"
	SkippedTasksTotal     GlobalHatchetMetric = "hatchet_skipped_tasks_total"
	CancelledTasksTotal   GlobalHatchetMetric = "hatchet_cancelled_tasks_total"
)

var (
	QueueInvocations = promauto.NewCounter(prometheus.CounterOpts{
		Name: string(QueueInvocationsTotal),
		Help: "The total number of invocations of the queuer function",
	})

	CreatedTasks = promauto.NewCounter(prometheus.CounterOpts{
		Name: string(CreatedTasksTotal),
		Help: "The total number of tasks created",
	})

	RetriedTasks = promauto.NewCounter(prometheus.CounterOpts{
		Name: string(RetriedTasksTotal),
		Help: "The total number of tasks retried",
	})

	SucceededTasks = promauto.NewCounter(prometheus.CounterOpts{
		Name: string(SucceededTasksTotal),
		Help: "The total number of tasks that succeeded",
	})

	FailedTasks = promauto.NewCounter(prometheus.CounterOpts{
		Name: string(FailedTasksTotal),
		Help: "The total number of tasks that failed (in a final state, not including retries)",
	})

	SkippedTasks = promauto.NewCounter(prometheus.CounterOpts{
		Name: string(SkippedTasksTotal),
		Help: "The total number of tasks that were skipped",
	})

	CancelledTasks = promauto.NewCounter(prometheus.CounterOpts{
		Name: string(CancelledTasksTotal),
		Help: "The total number of tasks cancelled",
	})
)
