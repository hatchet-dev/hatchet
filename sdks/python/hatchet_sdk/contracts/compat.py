from __future__ import annotations

import warnings
from typing import Iterable

from hatchet_sdk.contracts import dispatcher_pb2, events_pb2, workflows_pb2
from hatchet_sdk.contracts.v1 import workflows_pb2 as v1_workflows_pb2


def _add_aliases(cls: type, aliases: Iterable[tuple[str, str]]) -> None:
    for old_name, new_name in aliases:
        if hasattr(cls, old_name) or not hasattr(cls, new_name):
            continue

        def getter(self, _new_name=new_name, _old_name=old_name):
            warnings.warn(
                f"'{_old_name}' is deprecated, use '{_new_name}' instead.",
                DeprecationWarning,
                stacklevel=2,
            )
            return getattr(self, _new_name)

        def setter(self, value, _new_name=new_name, _old_name=old_name):
            warnings.warn(
                f"'{_old_name}' is deprecated, use '{_new_name}' instead.",
                DeprecationWarning,
                stacklevel=2,
            )
            setattr(self, _new_name, value)

        setattr(cls, old_name, property(getter, setter))


def apply_proto_aliases() -> None:
    _add_aliases(
        dispatcher_pb2.SubscribeToWorkflowRunsRequest,
        [("workflowRunId", "workflow_run_id")],
    )
    _add_aliases(
        dispatcher_pb2.SubscribeToWorkflowEventsRequest,
        [
            ("workflowRunId", "workflow_run_id"),
            ("additionalMetaKey", "additional_meta_key"),
            ("additionalMetaValue", "additional_meta_value"),
        ],
    )
    _add_aliases(
        dispatcher_pb2.WorkflowEvent,
        [
            ("workflowRunId", "workflow_run_id"),
            ("resourceType", "resource_type"),
            ("eventType", "event_type"),
            ("resourceId", "resource_id"),
            ("eventTimestamp", "event_timestamp"),
            ("eventPayload", "event_payload"),
            ("stepRetries", "step_retries"),
            ("retryCount", "retry_count"),
            ("eventIndex", "event_index"),
        ],
    )
    _add_aliases(
        dispatcher_pb2.WorkflowRunEvent,
        [
            ("workflowRunId", "workflow_run_id"),
            ("eventType", "event_type"),
            ("eventTimestamp", "event_timestamp"),
        ],
    )
    _add_aliases(
        dispatcher_pb2.StepRunResult,
        [
            ("stepRunId", "step_run_id"),
            ("stepReadableId", "step_readable_id"),
            ("jobRunId", "job_run_id"),
        ],
    )
    _add_aliases(
        dispatcher_pb2.WorkerLabels,
        [("strValue", "str_value"), ("intValue", "int_value")],
    )
    _add_aliases(
        dispatcher_pb2.RuntimeInfo,
        [("sdkVersion", "sdk_version"), ("languageVersion", "language_version")],
    )
    _add_aliases(
        dispatcher_pb2.WorkerRegisterRequest,
        [("workerName", "worker_name"), ("maxRuns", "slots"), ("runtimeInfo", "runtime_info")],
    )
    _add_aliases(
        dispatcher_pb2.WorkerRegisterResponse,
        [("workerId", "worker_id")],
    )
    _add_aliases(
        dispatcher_pb2.AssignedAction,
        [
            ("tenantId", "tenant_id"),
            ("workflowRunId", "workflow_run_id"),
            ("getGroupKeyRunId", "get_group_key_run_id"),
            ("jobId", "job_id"),
            ("jobName", "job_name"),
            ("jobRunId", "job_run_id"),
            ("stepId", "task_id"),
            ("stepRunId", "task_run_id"),
            ("actionId", "action_id"),
            ("actionType", "action_type"),
            ("actionPayload", "action_payload"),
            ("stepName", "step_name"),
            ("retryCount", "retry_count"),
            ("additionalMetadata", "additional_metadata"),
            ("childWorkflowIndex", "child_workflow_index"),
            ("childWorkflowKey", "child_workflow_key"),
            ("parentWorkflowRunId", "parent_workflow_run_id"),
            ("workflowId", "workflow_id"),
            ("workflowVersionId", "workflow_version_id"),
        ],
    )
    _add_aliases(
        dispatcher_pb2.StepActionEvent,
        [
            ("workerId", "worker_id"),
            ("jobId", "job_id"),
            ("jobRunId", "job_run_id"),
            ("stepId", "task_id"),
            ("stepRunId", "task_run_id"),
            ("actionId", "action_id"),
            ("eventTimestamp", "event_timestamp"),
            ("eventType", "event_type"),
            ("eventPayload", "event_payload"),
            ("retryCount", "retry_count"),
            ("shouldNotRetry", "should_not_retry"),
        ],
    )
    _add_aliases(
        dispatcher_pb2.HeartbeatRequest,
        [("workerId", "worker_id"), ("heartbeatAt", "heartbeat_at")],
    )
    _add_aliases(
        dispatcher_pb2.WorkerListenRequest,
        [("workerId", "worker_id")],
    )
    _add_aliases(
        dispatcher_pb2.WorkerUnsubscribeRequest,
        [("workerId", "worker_id")],
    )
    _add_aliases(
        dispatcher_pb2.RefreshTimeoutRequest,
        [("stepRunId", "step_run_id"), ("incrementTimeoutBy", "increment_timeout_by")],
    )
    _add_aliases(
        dispatcher_pb2.ReleaseSlotRequest,
        [("stepRunId", "step_run_id")],
    )

    _add_aliases(
        events_pb2.Event,
        [
            ("eventId", "event_id"),
            ("eventTimestamp", "event_timestamp"),
            ("additionalMetadata", "additional_metadata"),
        ],
    )
    _add_aliases(
        events_pb2.PushEventRequest,
        [("eventTimestamp", "event_timestamp"), ("additionalMetadata", "additional_metadata")],
    )
    _add_aliases(
        events_pb2.PutLogRequest,
        [
            ("stepRunId", "task_run_id"),
            ("createdAt", "created_at"),
            ("taskRetryCount", "task_retry_count"),
        ],
    )
    _add_aliases(
        events_pb2.PutStreamEventRequest,
        [
            ("stepRunId", "task_run_id"),
            ("createdAt", "created_at"),
            ("eventIndex", "event_index"),
        ],
    )
    _add_aliases(
        events_pb2.ReplayEventRequest,
        [("eventId", "event_id")],
    )

    _add_aliases(
        workflows_pb2.TriggerWorkflowRequest,
        [("parent_step_run_id", "parent_task_run_id")],
    )
    _add_aliases(
        workflows_pb2.ScheduleWorkflowRequest,
        [("parent_step_run_id", "parent_task_run_id")],
    )
    _add_aliases(
        v1_workflows_pb2.DesiredWorkerLabels,
        [("strValue", "str_value"), ("intValue", "int_value")],
    )
