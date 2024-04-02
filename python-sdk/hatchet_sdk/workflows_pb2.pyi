from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from typing import ClassVar as _ClassVar, Iterable as _Iterable, Mapping as _Mapping, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ConcurrencyLimitStrategy(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    CANCEL_IN_PROGRESS: _ClassVar[ConcurrencyLimitStrategy]
    DROP_NEWEST: _ClassVar[ConcurrencyLimitStrategy]
    QUEUE_NEWEST: _ClassVar[ConcurrencyLimitStrategy]
    GROUP_ROUND_ROBIN: _ClassVar[ConcurrencyLimitStrategy]

class RateLimitDuration(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    SECOND: _ClassVar[RateLimitDuration]
    MINUTE: _ClassVar[RateLimitDuration]
    HOUR: _ClassVar[RateLimitDuration]
CANCEL_IN_PROGRESS: ConcurrencyLimitStrategy
DROP_NEWEST: ConcurrencyLimitStrategy
QUEUE_NEWEST: ConcurrencyLimitStrategy
GROUP_ROUND_ROBIN: ConcurrencyLimitStrategy
SECOND: RateLimitDuration
MINUTE: RateLimitDuration
HOUR: RateLimitDuration

class PutWorkflowRequest(_message.Message):
    __slots__ = ("opts",)
    OPTS_FIELD_NUMBER: _ClassVar[int]
    opts: CreateWorkflowVersionOpts
    def __init__(self, opts: _Optional[_Union[CreateWorkflowVersionOpts, _Mapping]] = ...) -> None: ...

class CreateWorkflowVersionOpts(_message.Message):
    __slots__ = ("name", "description", "version", "event_triggers", "cron_triggers", "scheduled_triggers", "jobs", "concurrency", "schedule_timeout", "cron_input")
    NAME_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    EVENT_TRIGGERS_FIELD_NUMBER: _ClassVar[int]
    CRON_TRIGGERS_FIELD_NUMBER: _ClassVar[int]
    SCHEDULED_TRIGGERS_FIELD_NUMBER: _ClassVar[int]
    JOBS_FIELD_NUMBER: _ClassVar[int]
    CONCURRENCY_FIELD_NUMBER: _ClassVar[int]
    SCHEDULE_TIMEOUT_FIELD_NUMBER: _ClassVar[int]
    CRON_INPUT_FIELD_NUMBER: _ClassVar[int]
    name: str
    description: str
    version: str
    event_triggers: _containers.RepeatedScalarFieldContainer[str]
    cron_triggers: _containers.RepeatedScalarFieldContainer[str]
    scheduled_triggers: _containers.RepeatedCompositeFieldContainer[_timestamp_pb2.Timestamp]
    jobs: _containers.RepeatedCompositeFieldContainer[CreateWorkflowJobOpts]
    concurrency: WorkflowConcurrencyOpts
    schedule_timeout: str
    cron_input: str
    def __init__(self, name: _Optional[str] = ..., description: _Optional[str] = ..., version: _Optional[str] = ..., event_triggers: _Optional[_Iterable[str]] = ..., cron_triggers: _Optional[_Iterable[str]] = ..., scheduled_triggers: _Optional[_Iterable[_Union[_timestamp_pb2.Timestamp, _Mapping]]] = ..., jobs: _Optional[_Iterable[_Union[CreateWorkflowJobOpts, _Mapping]]] = ..., concurrency: _Optional[_Union[WorkflowConcurrencyOpts, _Mapping]] = ..., schedule_timeout: _Optional[str] = ..., cron_input: _Optional[str] = ...) -> None: ...

class WorkflowConcurrencyOpts(_message.Message):
    __slots__ = ("action", "max_runs", "limit_strategy")
    ACTION_FIELD_NUMBER: _ClassVar[int]
    MAX_RUNS_FIELD_NUMBER: _ClassVar[int]
    LIMIT_STRATEGY_FIELD_NUMBER: _ClassVar[int]
    action: str
    max_runs: int
    limit_strategy: ConcurrencyLimitStrategy
    def __init__(self, action: _Optional[str] = ..., max_runs: _Optional[int] = ..., limit_strategy: _Optional[_Union[ConcurrencyLimitStrategy, str]] = ...) -> None: ...

class CreateWorkflowJobOpts(_message.Message):
    __slots__ = ("name", "description", "timeout", "steps")
    NAME_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    TIMEOUT_FIELD_NUMBER: _ClassVar[int]
    STEPS_FIELD_NUMBER: _ClassVar[int]
    name: str
    description: str
    timeout: str
    steps: _containers.RepeatedCompositeFieldContainer[CreateWorkflowStepOpts]
    def __init__(self, name: _Optional[str] = ..., description: _Optional[str] = ..., timeout: _Optional[str] = ..., steps: _Optional[_Iterable[_Union[CreateWorkflowStepOpts, _Mapping]]] = ...) -> None: ...

class CreateWorkflowStepOpts(_message.Message):
    __slots__ = ("readable_id", "action", "timeout", "inputs", "parents", "user_data", "retries", "rate_limits")
    READABLE_ID_FIELD_NUMBER: _ClassVar[int]
    ACTION_FIELD_NUMBER: _ClassVar[int]
    TIMEOUT_FIELD_NUMBER: _ClassVar[int]
    INPUTS_FIELD_NUMBER: _ClassVar[int]
    PARENTS_FIELD_NUMBER: _ClassVar[int]
    USER_DATA_FIELD_NUMBER: _ClassVar[int]
    RETRIES_FIELD_NUMBER: _ClassVar[int]
    RATE_LIMITS_FIELD_NUMBER: _ClassVar[int]
    readable_id: str
    action: str
    timeout: str
    inputs: str
    parents: _containers.RepeatedScalarFieldContainer[str]
    user_data: str
    retries: int
    rate_limits: _containers.RepeatedCompositeFieldContainer[CreateStepRateLimit]
    def __init__(self, readable_id: _Optional[str] = ..., action: _Optional[str] = ..., timeout: _Optional[str] = ..., inputs: _Optional[str] = ..., parents: _Optional[_Iterable[str]] = ..., user_data: _Optional[str] = ..., retries: _Optional[int] = ..., rate_limits: _Optional[_Iterable[_Union[CreateStepRateLimit, _Mapping]]] = ...) -> None: ...

class CreateStepRateLimit(_message.Message):
    __slots__ = ("key", "units")
    KEY_FIELD_NUMBER: _ClassVar[int]
    UNITS_FIELD_NUMBER: _ClassVar[int]
    key: str
    units: int
    def __init__(self, key: _Optional[str] = ..., units: _Optional[int] = ...) -> None: ...

class ListWorkflowsRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ScheduleWorkflowRequest(_message.Message):
    __slots__ = ("name", "schedules", "input", "parent_id", "parent_step_run_id", "child_index", "child_key")
    NAME_FIELD_NUMBER: _ClassVar[int]
    SCHEDULES_FIELD_NUMBER: _ClassVar[int]
    INPUT_FIELD_NUMBER: _ClassVar[int]
    PARENT_ID_FIELD_NUMBER: _ClassVar[int]
    PARENT_STEP_RUN_ID_FIELD_NUMBER: _ClassVar[int]
    CHILD_INDEX_FIELD_NUMBER: _ClassVar[int]
    CHILD_KEY_FIELD_NUMBER: _ClassVar[int]
    name: str
    schedules: _containers.RepeatedCompositeFieldContainer[_timestamp_pb2.Timestamp]
    input: str
    parent_id: str
    parent_step_run_id: str
    child_index: int
    child_key: str
    def __init__(self, name: _Optional[str] = ..., schedules: _Optional[_Iterable[_Union[_timestamp_pb2.Timestamp, _Mapping]]] = ..., input: _Optional[str] = ..., parent_id: _Optional[str] = ..., parent_step_run_id: _Optional[str] = ..., child_index: _Optional[int] = ..., child_key: _Optional[str] = ...) -> None: ...

class WorkflowVersion(_message.Message):
    __slots__ = ("id", "created_at", "updated_at", "version", "order", "workflow_id")
    ID_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    ORDER_FIELD_NUMBER: _ClassVar[int]
    WORKFLOW_ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    created_at: _timestamp_pb2.Timestamp
    updated_at: _timestamp_pb2.Timestamp
    version: str
    order: int
    workflow_id: str
    def __init__(self, id: _Optional[str] = ..., created_at: _Optional[_Union[_timestamp_pb2.Timestamp, _Mapping]] = ..., updated_at: _Optional[_Union[_timestamp_pb2.Timestamp, _Mapping]] = ..., version: _Optional[str] = ..., order: _Optional[int] = ..., workflow_id: _Optional[str] = ...) -> None: ...

class WorkflowTriggerEventRef(_message.Message):
    __slots__ = ("parent_id", "event_key")
    PARENT_ID_FIELD_NUMBER: _ClassVar[int]
    EVENT_KEY_FIELD_NUMBER: _ClassVar[int]
    parent_id: str
    event_key: str
    def __init__(self, parent_id: _Optional[str] = ..., event_key: _Optional[str] = ...) -> None: ...

class WorkflowTriggerCronRef(_message.Message):
    __slots__ = ("parent_id", "cron")
    PARENT_ID_FIELD_NUMBER: _ClassVar[int]
    CRON_FIELD_NUMBER: _ClassVar[int]
    parent_id: str
    cron: str
    def __init__(self, parent_id: _Optional[str] = ..., cron: _Optional[str] = ...) -> None: ...

class TriggerWorkflowRequest(_message.Message):
    __slots__ = ("name", "input", "parent_id", "parent_step_run_id", "child_index", "child_key")
    NAME_FIELD_NUMBER: _ClassVar[int]
    INPUT_FIELD_NUMBER: _ClassVar[int]
    PARENT_ID_FIELD_NUMBER: _ClassVar[int]
    PARENT_STEP_RUN_ID_FIELD_NUMBER: _ClassVar[int]
    CHILD_INDEX_FIELD_NUMBER: _ClassVar[int]
    CHILD_KEY_FIELD_NUMBER: _ClassVar[int]
    name: str
    input: str
    parent_id: str
    parent_step_run_id: str
    child_index: int
    child_key: str
    def __init__(self, name: _Optional[str] = ..., input: _Optional[str] = ..., parent_id: _Optional[str] = ..., parent_step_run_id: _Optional[str] = ..., child_index: _Optional[int] = ..., child_key: _Optional[str] = ...) -> None: ...

class TriggerWorkflowResponse(_message.Message):
    __slots__ = ("workflow_run_id",)
    WORKFLOW_RUN_ID_FIELD_NUMBER: _ClassVar[int]
    workflow_run_id: str
    def __init__(self, workflow_run_id: _Optional[str] = ...) -> None: ...

class PutRateLimitRequest(_message.Message):
    __slots__ = ("key", "limit", "duration")
    KEY_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    DURATION_FIELD_NUMBER: _ClassVar[int]
    key: str
    limit: int
    duration: RateLimitDuration
    def __init__(self, key: _Optional[str] = ..., limit: _Optional[int] = ..., duration: _Optional[_Union[RateLimitDuration, str]] = ...) -> None: ...

class PutRateLimitResponse(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...
