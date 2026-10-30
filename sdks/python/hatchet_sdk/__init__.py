# ruff: noqa: E402

import os

# Must be set before grpc is imported anywhere — the C extension reads this at load
# time and won't re-read it after. Disable by default; opt in via env var for
# setups that genuinely need fork support (e.g. Gunicorn prefork).
_fork_env = os.environ.get("HATCHET_CLIENT_GRPC_ENABLE_FORK_SUPPORT", "false").lower()
_grpc_fork = _fork_env in ("true", "1", "yes")
os.environ.setdefault("GRPC_ENABLE_FORK_SUPPORT", "true" if _grpc_fork else "false")
if _grpc_fork:
    os.environ.setdefault("GRPC_POLL_STRATEGY", "poll")

from hatchet_sdk.clients.rest.models.v1_task_summary import V1TaskSummary
from hatchet_sdk.clients.rest.models.v1_webhook_hmac_algorithm import (
    V1WebhookHMACAlgorithm,
)
from hatchet_sdk.clients.rest.models.v1_webhook_hmac_encoding import (
    V1WebhookHMACEncoding,
)
from hatchet_sdk.clients.rest.models.v1_webhook_source_name import V1WebhookSourceName
from hatchet_sdk.conditions import (
    Condition,
    OrGroup,
    ParentCondition,
    SleepCondition,
    UserEventCondition,
    or_,
)
from hatchet_sdk.config import (
    ClientConfig,
    ClientTLSConfig,
    EmbeddedHatchetConfig,
    HealthcheckConfig,
    HTTPMethod,
    OpenTelemetryConfig,
    TenacityConfig,
)
from hatchet_sdk.context.context import (
    Context,
    DurableContext,
    EventWaitResult,
    OrGroupResult,
    SleepResult,
)
from hatchet_sdk.exceptions import (
    BulkTriggerIdempotencyCollisionError,
    DedupeViolationError,
    EvictionNotSupportedError,
    FailedTaskRunExceptionGroup,
    IdempotencyCollisionError,
    NonDeterminismError,
    NonRetryableError,
    RetryAfterError,
    TaskRunError,
)
from hatchet_sdk.features.cel import CELEvaluationResult, CELFailure, CELSuccess
from hatchet_sdk.features.events import Event
from hatchet_sdk.features.runs import BulkCancelReplayOptions, RunFilter
from hatchet_sdk.features.streams import StreamEvent, StreamTopicMetadata
from hatchet_sdk.hatchet import Hatchet
from hatchet_sdk.runnables.eviction import EvictionPolicy
from hatchet_sdk.runnables.task import Depends, Task
from hatchet_sdk.runnables.types import (
    DefaultFilter,
    TaskDefaults,
)
from hatchet_sdk.runnables.workflow import Standalone, TaskRunRef, Workflow
from hatchet_sdk.serde import is_in_hatchet_serialization_context
from hatchet_sdk.types.concurrency import (
    ConcurrencyStrategy,
)
from hatchet_sdk.types.idempotency import (
    StatusBasedIdempotencyConfig,
    TTLBasedIdempotencyConfig,
)
from hatchet_sdk.types.labels import (
    DesiredWorkerLabel,
    WorkerLabel,
    WorkerLabelComparator,
)
from hatchet_sdk.types.priority import Priority
from hatchet_sdk.types.rate_limit import RateLimit
from hatchet_sdk.types.run_status import RunStatus
from hatchet_sdk.types.slot_types import SlotType
from hatchet_sdk.types.trigger import (
    BulkPushEventWithMetadata,
    WorkflowRunTriggerConfig,
)
from hatchet_sdk.utils.opentelemetry import OTelAttribute
from hatchet_sdk.utils.serde import remove_null_unicode_character
from hatchet_sdk.worker.worker import Worker, WorkerStatus
from hatchet_sdk.workflow_run import WorkflowRunRef

__all__ = [
    "BulkCancelReplayOptions",
    "BulkPushEventWithMetadata",
    "BulkTriggerIdempotencyCollisionError",
    "CELEvaluationResult",
    "CELFailure",
    "CELSuccess",
    "ClientConfig",
    "ClientTLSConfig",
    "ConcurrencyStrategy",
    "Condition",
    "Context",
    "DedupeViolationError",
    "DefaultFilter",
    "Depends",
    "DesiredWorkerLabel",
    "DurableContext",
    "EmbeddedHatchetConfig",
    "Event",
    "EventWaitResult",
    "EvictionNotSupportedError",
    "EvictionPolicy",
    "FailedTaskRunExceptionGroup",
    "HTTPMethod",
    "Hatchet",
    "HealthcheckConfig",
    "IdempotencyCollisionError",
    "NonDeterminismError",
    "NonRetryableError",
    "OTelAttribute",
    "OpenTelemetryConfig",
    "OrGroup",
    "OrGroupResult",
    "ParentCondition",
    "Priority",
    "RateLimit",
    "RetryAfterError",
    "RunFilter",
    "RunStatus",
    "SleepCondition",
    "SleepResult",
    "SlotType",
    "Standalone",
    "StatusBasedIdempotencyConfig",
    "StreamEvent",
    "StreamTopicMetadata",
    "TTLBasedIdempotencyConfig",
    "Task",
    "TaskDefaults",
    "TaskRunError",
    "TaskRunRef",
    "TenacityConfig",
    "UserEventCondition",
    "V1TaskSummary",
    "V1WebhookHMACAlgorithm",
    "V1WebhookHMACEncoding",
    "V1WebhookSourceName",
    "Worker",
    "WorkerLabel",
    "WorkerLabelComparator",
    "WorkerStatus",
    "Workflow",
    "WorkflowRunRef",
    "WorkflowRunTriggerConfig",
    "is_in_hatchet_serialization_context",
    "or_",
    "remove_null_unicode_character",
]
