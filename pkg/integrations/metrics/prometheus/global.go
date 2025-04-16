package prometheus

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type GlobalHatchetMetric string

const (
	QueueInvocationsCount                GlobalHatchetMetric = "queue_invocations"
	CreatedTasksCount                    GlobalHatchetMetric = "created_tasks_count"
	RetriedTasksCount                    GlobalHatchetMetric = "retried_tasks_count"
	SucceededTasksCount                  GlobalHatchetMetric = "succeeded_tasks_count"
	FailedTasksCount                     GlobalHatchetMetric = "failed_tasks_count"
	SkippedTasksCount                    GlobalHatchetMetric = "skipped_tasks_count"
	CancelledTasksCount                  GlobalHatchetMetric = "cancelled_tasks_count"
	AssignedTasksCount                   GlobalHatchetMetric = "assigned_tasks_count"
	SchedulingTimedOutCount              GlobalHatchetMetric = "scheduling_timed_out_count"
	RateLimitedCount                     GlobalHatchetMetric = "rate_limited_count"
	QueuedToAssignedCount                GlobalHatchetMetric = "queued_to_assigned_count"
	QueuedToAssignedTimeNanosecondsCount GlobalHatchetMetric = "queued_to_assigned_nanoseconds_count"
)

var (
	QueueInvocations = promauto.NewCounter(prometheus.CounterOpts{
		Name: string(QueueInvocationsCount),
		Help: "The total number of invocations of the queuer function",
	})

	CreatedTasks = promauto.NewCounter(prometheus.CounterOpts{
		Name: string(CreatedTasksCount),
		Help: "The total number of tasks created",
	})

	RetriedTasks = promauto.NewCounter(prometheus.CounterOpts{
		Name: string(RetriedTasksCount),
		Help: "The total number of tasks retried",
	})

	SucceededTasks = promauto.NewCounter(prometheus.CounterOpts{
		Name: string(SucceededTasksCount),
		Help: "The total number of tasks that succeeded",
	})

	FailedTasks = promauto.NewCounter(prometheus.CounterOpts{
		Name: string(FailedTasksCount),
		Help: "The total number of tasks that failed (in a final state, not including retries)",
	})

	SkippedTasks = promauto.NewCounter(prometheus.CounterOpts{
		Name: string(SkippedTasksCount),
		Help: "The total number of tasks that were skipped",
	})

	CancelledTasks = promauto.NewCounter(prometheus.CounterOpts{
		Name: string(CancelledTasksCount),
		Help: "The total number of tasks cancelled",
	})

	AssignedTasks = promauto.NewCounter(prometheus.CounterOpts{
		Name: string(AssignedTasksCount),
		Help: "The total number of tasks assigned to a worker",
	})

	SchedulingTimedOut = promauto.NewCounter(prometheus.CounterOpts{
		Name: string(SchedulingTimedOutCount),
		Help: "The total number of tasks that timed out while waiting to be scheduled",
	})

	RateLimited = promauto.NewCounter(prometheus.CounterOpts{
		Name: string(RateLimitedCount),
		Help: "The total number of tasks that were rate limited",
	})

	QueuedToAssigned = promauto.NewCounter(prometheus.CounterOpts{
		Name: string(QueuedToAssignedCount),
		Help: "The total number of unique tasks that were queued and later got assigned to a worker",
	})

	QueuedToAssignedTimeNanoseconds = promauto.NewCounter(prometheus.CounterOpts{
		Name: string(QueuedToAssignedTimeNanosecondsCount),
		Help: "The total time in nanoseconds spent in the queue before being assigned to a worker",
	})
)
