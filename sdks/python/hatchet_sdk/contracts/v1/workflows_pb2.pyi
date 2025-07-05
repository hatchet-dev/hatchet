from google.protobuf import timestamp_pb2 as _timestamp_pb2
from hatchet_sdk.contracts.v1.shared import condition_pb2 as _condition_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from typing import ClassVar as _ClassVar, Iterable as _Iterable, Mapping as _Mapping, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class StickyStrategy(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    SOFT: _ClassVar[StickyStrategy]
    HARD: _ClassVar[StickyStrategy]

class RateLimitDuration(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    SECOND: _ClassVar[RateLimitDuration]
    MINUTE: _ClassVar[RateLimitDuration]
    HOUR: _ClassVar[RateLimitDuration]
    DAY: _ClassVar[RateLimitDuration]
    WEEK: _ClassVar[RateLimitDuration]
    MONTH: _ClassVar[RateLimitDuration]
    YEAR: _ClassVar[RateLimitDuration]

class ConcurrencyLimitStrategy(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    CANCEL_IN_PROGRESS: _ClassVar[ConcurrencyLimitStrategy]
    DROP_NEWEST: _ClassVar[ConcurrencyLimitStrategy]
    QUEUE_NEWEST: _ClassVar[ConcurrencyLimitStrategy]
    GROUP_ROUND_ROBIN: _ClassVar[ConcurrencyLimitStrategy]
    CANCEL_NEWEST: _ClassVar[ConcurrencyLimitStrategy]

class WorkerLabelComparator(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    EQUAL: _ClassVar[WorkerLabelComparator]
    NOT_EQUAL: _ClassVar[WorkerLabelComparator]
    GREATER_THAN: _ClassVar[WorkerLabelComparator]
    GREATER_THAN_OR_EQUAL: _ClassVar[WorkerLabelComparator]
    LESS_THAN: _ClassVar[WorkerLabelComparator]
    LESS_THAN_OR_EQUAL: _ClassVar[WorkerLabelComparator]
SOFT: StickyStrategy
HARD: StickyStrategy
SECOND: RateLimitDuration
MINUTE: RateLimitDuration
HOUR: RateLimitDuration
DAY: RateLimitDuration
WEEK: RateLimitDuration
MONTH: RateLimitDuration
YEAR: RateLimitDuration
CANCEL_IN_PROGRESS: ConcurrencyLimitStrategy
DROP_NEWEST: ConcurrencyLimitStrategy
QUEUE_NEWEST: ConcurrencyLimitStrategy
GROUP_ROUND_ROBIN: ConcurrencyLimitStrategy
CANCEL_NEWEST: ConcurrencyLimitStrategy
EQUAL: WorkerLabelComparator
NOT_EQUAL: WorkerLabelComparator
GREATER_THAN: WorkerLabelComparator
GREATER_THAN_OR_EQUAL: WorkerLabelComparator
LESS_THAN: WorkerLabelComparator
LESS_THAN_OR_EQUAL: WorkerLabelComparator

class CancelTasksRequest(_message.Message):
    __slots__ = ("externalIds", "filter")
    EXTERNALIDS_FIELD_NUMBER: _ClassVar[int]
    FILTER_FIELD_NUMBER: _ClassVar[int]
    externalIds: _containers.RepeatedScalarFieldContainer[str]
    filter: TasksFilter
    def __init__(self, externalIds: _Optional[_Iterable[str]] = ..., filter: _Optional[_Union[TasksFilter, _Mapping]] = ...) -> None: ...

class ReplayTasksRequest(_message.Message):
    __slots__ = ("externalIds", "filter")
    EXTERNALIDS_FIELD_NUMBER: _ClassVar[int]
    FILTER_FIELD_NUMBER: _ClassVar[int]
    externalIds: _containers.RepeatedScalarFieldContainer[str]
    filter: TasksFilter
    def __init__(self, externalIds: _Optional[_Iterable[str]] = ..., filter: _Optional[_Union[TasksFilter, _Mapping]] = ...) -> None: ...

class TasksFilter(_message.Message):
    __slots__ = ("statuses", "since", "until", "workflow_ids", "additional_metadata")
    STATUSES_FIELD_NUMBER: _ClassVar[int]
    SINCE_FIELD_NUMBER: _ClassVar[int]
    UNTIL_FIELD_NUMBER: _ClassVar[int]
    WORKFLOW_IDS_FIELD_NUMBER: _ClassVar[int]
    ADDITIONAL_METADATA_FIELD_NUMBER: _ClassVar[int]
    statuses: _containers.RepeatedScalarFieldContainer[str]
    since: _timestamp_pb2.Timestamp
    until: _timestamp_pb2.Timestamp
    workflow_ids: _containers.RepeatedScalarFieldContainer[str]
    additional_metadata: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, statuses: _Optional[_Iterable[str]] = ..., since: _Optional[_Union[_timestamp_pb2.Timestamp, _Mapping]] = ..., until: _Optional[_Union[_timestamp_pb2.Timestamp, _Mapping]] = ..., workflow_ids: _Optional[_Iterable[str]] = ..., additional_metadata: _Optional[_Iterable[str]] = ...) -> None: ...

class CancelTasksResponse(_message.Message):
    __slots__ = ("cancelled_tasks",)
    CANCELLED_TASKS_FIELD_NUMBER: _ClassVar[int]
    cancelled_tasks: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, cancelled_tasks: _Optional[_Iterable[str]] = ...) -> None: ...

class ReplayTasksResponse(_message.Message):
    __slots__ = ("replayed_tasks",)
    REPLAYED_TASKS_FIELD_NUMBER: _ClassVar[int]
    replayed_tasks: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, replayed_tasks: _Optional[_Iterable[str]] = ...) -> None: ...

class TriggerWorkflowRunRequest(_message.Message):
    __slots__ = ("workflow_name", "input", "additional_metadata", "priority")
    WORKFLOW_NAME_FIELD_NUMBER: _ClassVar[int]
    INPUT_FIELD_NUMBER: _ClassVar[int]
    ADDITIONAL_METADATA_FIELD_NUMBER: _ClassVar[int]
    PRIORITY_FIELD_NUMBER: _ClassVar[int]
    workflow_name: str
    input: bytes
    additional_metadata: bytes
    priority: int
    def __init__(self, workflow_name: _Optional[str] = ..., input: _Optional[bytes] = ..., additional_metadata: _Optional[bytes] = ..., priority: _Optional[int] = ...) -> None: ...

class TriggerWorkflowRunResponse(_message.Message):
    __slots__ = ("external_id",)
    EXTERNAL_ID_FIELD_NUMBER: _ClassVar[int]
    external_id: str
    def __init__(self, external_id: _Optional[str] = ...) -> None: ...

class CreateWorkflowVersionRequest(_message.Message):
    __slots__ = ("name", "description", "version", "event_triggers", "cron_triggers", "tasks", "concurrency", "cron_input", "on_failure_task", "sticky", "default_priority", "concurrency_arr", "default_filters")
    NAME_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    EVENT_TRIGGERS_FIELD_NUMBER: _ClassVar[int]
    CRON_TRIGGERS_FIELD_NUMBER: _ClassVar[int]
    TASKS_FIELD_NUMBER: _ClassVar[int]
    CONCURRENCY_FIELD_NUMBER: _ClassVar[int]
    CRON_INPUT_FIELD_NUMBER: _ClassVar[int]
    ON_FAILURE_TASK_FIELD_NUMBER: _ClassVar[int]
    STICKY_FIELD_NUMBER: _ClassVar[int]
    DEFAULT_PRIORITY_FIELD_NUMBER: _ClassVar[int]
    CONCURRENCY_ARR_FIELD_NUMBER: _ClassVar[int]
    DEFAULT_FILTERS_FIELD_NUMBER: _ClassVar[int]
    name: str
    description: str
    version: str
    event_triggers: _containers.RepeatedScalarFieldContainer[str]
    cron_triggers: _containers.RepeatedScalarFieldContainer[str]
    tasks: _containers.RepeatedCompositeFieldContainer[CreateTaskOpts]
    concurrency: Concurrency
    cron_input: str
    on_failure_task: CreateTaskOpts
    sticky: StickyStrategy
    default_priority: int
    concurrency_arr: _containers.RepeatedCompositeFieldContainer[Concurrency]
    default_filters: _containers.RepeatedCompositeFieldContainer[DefaultFilter]
    def __init__(self, name: _Optional[str] = ..., description: _Optional[str] = ..., version: _Optional[str] = ..., event_triggers: _Optional[_Iterable[str]] = ..., cron_triggers: _Optional[_Iterable[str]] = ..., tasks: _Optional[_Iterable[_Union[CreateTaskOpts, _Mapping]]] = ..., concurrency: _Optional[_Union[Concurrency, _Mapping]] = ..., cron_input: _Optional[str] = ..., on_failure_task: _Optional[_Union[CreateTaskOpts, _Mapping]] = ..., sticky: _Optional[_Union[StickyStrategy, str]] = ..., default_priority: _Optional[int] = ..., concurrency_arr: _Optional[_Iterable[_Union[Concurrency, _Mapping]]] = ..., default_filters: _Optional[_Iterable[_Union[DefaultFilter, _Mapping]]] = ...) -> None: ...

class DefaultFilter(_message.Message):
    __slots__ = ("expression", "scope", "payload")
    EXPRESSION_FIELD_NUMBER: _ClassVar[int]
    SCOPE_FIELD_NUMBER: _ClassVar[int]
    PAYLOAD_FIELD_NUMBER: _ClassVar[int]
    expression: str
    scope: str
    payload: bytes
    def __init__(self, expression: _Optional[str] = ..., scope: _Optional[str] = ..., payload: _Optional[bytes] = ...) -> None: ...

class Concurrency(_message.Message):
    __slots__ = ("expression", "max_runs", "limit_strategy")
    EXPRESSION_FIELD_NUMBER: _ClassVar[int]
    MAX_RUNS_FIELD_NUMBER: _ClassVar[int]
    LIMIT_STRATEGY_FIELD_NUMBER: _ClassVar[int]
    expression: str
    max_runs: int
    limit_strategy: ConcurrencyLimitStrategy
    def __init__(self, expression: _Optional[str] = ..., max_runs: _Optional[int] = ..., limit_strategy: _Optional[_Union[ConcurrencyLimitStrategy, str]] = ...) -> None: ...

class DesiredWorkerLabels(_message.Message):
    __slots__ = ("strValue", "intValue", "required", "comparator", "weight")
    STRVALUE_FIELD_NUMBER: _ClassVar[int]
    INTVALUE_FIELD_NUMBER: _ClassVar[int]
    REQUIRED_FIELD_NUMBER: _ClassVar[int]
    COMPARATOR_FIELD_NUMBER: _ClassVar[int]
    WEIGHT_FIELD_NUMBER: _ClassVar[int]
    strValue: str
    intValue: int
    required: bool
    comparator: WorkerLabelComparator
    weight: int
    def __init__(self, strValue: _Optional[str] = ..., intValue: _Optional[int] = ..., required: bool = ..., comparator: _Optional[_Union[WorkerLabelComparator, str]] = ..., weight: _Optional[int] = ...) -> None: ...

class CreateTaskOpts(_message.Message):
    __slots__ = ("readable_id", "action", "timeout", "inputs", "parents", "retries", "rate_limits", "worker_labels", "backoff_factor", "backoff_max_seconds", "concurrency", "conditions", "schedule_timeout")
    class WorkerLabelsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: DesiredWorkerLabels
        def __init__(self, key: _Optional[str] = ..., value: _Optional[_Union[DesiredWorkerLabels, _Mapping]] = ...) -> None: ...
    READABLE_ID_FIELD_NUMBER: _ClassVar[int]
    ACTION_FIELD_NUMBER: _ClassVar[int]
    TIMEOUT_FIELD_NUMBER: _ClassVar[int]
    INPUTS_FIELD_NUMBER: _ClassVar[int]
    PARENTS_FIELD_NUMBER: _ClassVar[int]
    RETRIES_FIELD_NUMBER: _ClassVar[int]
    RATE_LIMITS_FIELD_NUMBER: _ClassVar[int]
    WORKER_LABELS_FIELD_NUMBER: _ClassVar[int]
    BACKOFF_FACTOR_FIELD_NUMBER: _ClassVar[int]
    BACKOFF_MAX_SECONDS_FIELD_NUMBER: _ClassVar[int]
    CONCURRENCY_FIELD_NUMBER: _ClassVar[int]
    CONDITIONS_FIELD_NUMBER: _ClassVar[int]
    SCHEDULE_TIMEOUT_FIELD_NUMBER: _ClassVar[int]
    readable_id: str
    action: str
    timeout: str
    inputs: str
    parents: _containers.RepeatedScalarFieldContainer[str]
    retries: int
    rate_limits: _containers.RepeatedCompositeFieldContainer[CreateTaskRateLimit]
    worker_labels: _containers.MessageMap[str, DesiredWorkerLabels]
    backoff_factor: float
    backoff_max_seconds: int
    concurrency: _containers.RepeatedCompositeFieldContainer[Concurrency]
    conditions: _condition_pb2.TaskConditions
    schedule_timeout: str
    def __init__(self, readable_id: _Optional[str] = ..., action: _Optional[str] = ..., timeout: _Optional[str] = ..., inputs: _Optional[str] = ..., parents: _Optional[_Iterable[str]] = ..., retries: _Optional[int] = ..., rate_limits: _Optional[_Iterable[_Union[CreateTaskRateLimit, _Mapping]]] = ..., worker_labels: _Optional[_Mapping[str, DesiredWorkerLabels]] = ..., backoff_factor: _Optional[float] = ..., backoff_max_seconds: _Optional[int] = ..., concurrency: _Optional[_Iterable[_Union[Concurrency, _Mapping]]] = ..., conditions: _Optional[_Union[_condition_pb2.TaskConditions, _Mapping]] = ..., schedule_timeout: _Optional[str] = ...) -> None: ...

class CreateTaskRateLimit(_message.Message):
    __slots__ = ("key", "units", "key_expr", "units_expr", "limit_values_expr", "duration")
    KEY_FIELD_NUMBER: _ClassVar[int]
    UNITS_FIELD_NUMBER: _ClassVar[int]
    KEY_EXPR_FIELD_NUMBER: _ClassVar[int]
    UNITS_EXPR_FIELD_NUMBER: _ClassVar[int]
    LIMIT_VALUES_EXPR_FIELD_NUMBER: _ClassVar[int]
    DURATION_FIELD_NUMBER: _ClassVar[int]
    key: str
    units: int
    key_expr: str
    units_expr: str
    limit_values_expr: str
    duration: RateLimitDuration
    def __init__(self, key: _Optional[str] = ..., units: _Optional[int] = ..., key_expr: _Optional[str] = ..., units_expr: _Optional[str] = ..., limit_values_expr: _Optional[str] = ..., duration: _Optional[_Union[RateLimitDuration, str]] = ...) -> None: ...

class CreateWorkflowVersionResponse(_message.Message):
    __slots__ = ("id", "workflow_id")
    ID_FIELD_NUMBER: _ClassVar[int]
    WORKFLOW_ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    workflow_id: str
    def __init__(self, id: _Optional[str] = ..., workflow_id: _Optional[str] = ...) -> None: ...
