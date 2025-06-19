package prometheus

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type GlobalHatchetMetric string

const (
	QueueInvocationsTotal       GlobalHatchetMetric = "hatchet_queue_invocations_total"
	CreatedTasksTotal           GlobalHatchetMetric = "hatchet_created_tasks_total"
	RetriedTasksTotal           GlobalHatchetMetric = "hatchet_retried_tasks_total"
	SucceededTasksTotal         GlobalHatchetMetric = "hatchet_succeeded_tasks_total"
	FailedTasksTotal            GlobalHatchetMetric = "hatchet_failed_tasks_total"
	SkippedTasksTotal           GlobalHatchetMetric = "hatchet_skipped_tasks_total"
	CancelledTasksTotal         GlobalHatchetMetric = "hatchet_cancelled_tasks_total"
	AssignedTasksTotal          GlobalHatchetMetric = "hatchet_assigned_tasks_total"
	SchedulingTimedOutTotal     GlobalHatchetMetric = "hatchet_scheduling_timed_out_total"
	RateLimitedTotal            GlobalHatchetMetric = "hatchet_rate_limited_total"
	QueuedToAssignedTotal       GlobalHatchetMetric = "hatchet_queued_to_assigned_total"
	QueuedToAssignedTimeSeconds GlobalHatchetMetric = "hatchet_queued_to_assigned_seconds"

	TenantFinishedWorkflowsTotal GlobalHatchetMetric = "tenant_finished_workflows_total"
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

	AssignedTasks = promauto.NewCounter(prometheus.CounterOpts{
		Name: string(AssignedTasksTotal),
		Help: "The total number of tasks assigned to a worker",
	})

	SchedulingTimedOut = promauto.NewCounter(prometheus.CounterOpts{
		Name: string(SchedulingTimedOutTotal),
		Help: "The total number of tasks that timed out while waiting to be scheduled",
	})

	RateLimited = promauto.NewCounter(prometheus.CounterOpts{
		Name: string(RateLimitedTotal),
		Help: "The total number of tasks that were rate limited",
	})

	QueuedToAssigned = promauto.NewCounter(prometheus.CounterOpts{
		Name: string(QueuedToAssignedTotal),
		Help: "The total number of unique tasks that were queued and later got assigned to a worker",
	})

	QueuedToAssignedTimeBuckets = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    string(QueuedToAssignedTimeSeconds),
		Help:    "Buckets of time in seconds spent in the queue before being assigned to a worker",
		Buckets: []float64{0.01, 0.02, 0.05, 0.1, 0.5, 1, 2, 5, 15},
	})

	TenantFinishedWorkflows = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: string(TenantFinishedWorkflowsTotal),
		Help: "The total number of finished workflow runs",
	}, []string{"tenant_id", "workflow_name", "status", "duration"})
)
