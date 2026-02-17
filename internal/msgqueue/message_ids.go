package msgqueue

// Message ID constants for tenant messages
const (
	MsgIDCancelTasks                  = "cancel-tasks"
	MsgIDDurableCallbackCompleted     = "durable-callback-completed"
	MsgIDCELEvaluationFailure         = "cel-evaluation-failure"
	MsgIDCheckTenantQueue             = "check-tenant-queue"
	MsgIDNewWorker                    = "new-worker"
	MsgIDNewQueue                     = "new-queue"
	MsgIDNewConcurrencyStrategy       = "new-concurrency-strategy"
	MsgIDCreateMonitoringEvent        = "create-monitoring-event"
	MsgIDCreatedDAG                   = "created-dag"
	MsgIDCreatedEventTrigger          = "created-event-trigger"
	MsgIDCreatedTask                  = "created-task"
	MsgIDFailedWebhookValidation      = "failed-webhook-validation"
	MsgIDInternalEvent                = "internal-event"
	MsgIDOffloadPayload               = "offload-payload"
	MsgIDReplayTasks                  = "replay-tasks"
	MsgIDTaskAssignedBulk             = "task-assigned-bulk"
	MsgIDTaskCancelled                = "task-cancelled"
	MsgIDTaskCompleted                = "task-completed"
	MsgIDTaskFailed                   = "task-failed"
	MsgIDTaskStreamEvent              = "task-stream-event"
	MsgIDTaskTrigger                  = "task-trigger"
	MsgIDUserEvent                    = "user-event"
	MsgIDWorkflowRunFinished          = "workflow-run-finished"
	MsgIDWorkflowRunFinishedCandidate = "workflow-run-finished-candidate"
)
