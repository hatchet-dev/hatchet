from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf import wrappers_pb2 as _wrappers_pb2
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
CANCEL_IN_PROGRESS: ConcurrencyLimitStrategy
DROP_NEWEST: ConcurrencyLimitStrategy
QUEUE_NEWEST: ConcurrencyLimitStrategy

class PutWorkflowRequest(_message.Message):
    __slots__ = ("opts",)
    OPTS_FIELD_NUMBER: _ClassVar[int]
    opts: CreateWorkflowVersionOpts
    def __init__(self, opts: _Optional[_Union[CreateWorkflowVersionOpts, _Mapping]] = ...) -> None: ...

class CreateWorkflowVersionOpts(_message.Message):
    __slots__ = ("name", "description", "version", "event_triggers", "cron_triggers", "scheduled_triggers", "jobs", "concurrency")
    NAME_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    EVENT_TRIGGERS_FIELD_NUMBER: _ClassVar[int]
    CRON_TRIGGERS_FIELD_NUMBER: _ClassVar[int]
    SCHEDULED_TRIGGERS_FIELD_NUMBER: _ClassVar[int]
    JOBS_FIELD_NUMBER: _ClassVar[int]
    CONCURRENCY_FIELD_NUMBER: _ClassVar[int]
    name: str
    description: str
    version: str
    event_triggers: _containers.RepeatedScalarFieldContainer[str]
    cron_triggers: _containers.RepeatedScalarFieldContainer[str]
    scheduled_triggers: _containers.RepeatedCompositeFieldContainer[_timestamp_pb2.Timestamp]
    jobs: _containers.RepeatedCompositeFieldContainer[CreateWorkflowJobOpts]
    concurrency: WorkflowConcurrencyOpts
    def __init__(self, name: _Optional[str] = ..., description: _Optional[str] = ..., version: _Optional[str] = ..., event_triggers: _Optional[_Iterable[str]] = ..., cron_triggers: _Optional[_Iterable[str]] = ..., scheduled_triggers: _Optional[_Iterable[_Union[_timestamp_pb2.Timestamp, _Mapping]]] = ..., jobs: _Optional[_Iterable[_Union[CreateWorkflowJobOpts, _Mapping]]] = ..., concurrency: _Optional[_Union[WorkflowConcurrencyOpts, _Mapping]] = ...) -> None: ...

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
    __slots__ = ("readable_id", "action", "timeout", "inputs", "parents")
    READABLE_ID_FIELD_NUMBER: _ClassVar[int]
    ACTION_FIELD_NUMBER: _ClassVar[int]
    TIMEOUT_FIELD_NUMBER: _ClassVar[int]
    INPUTS_FIELD_NUMBER: _ClassVar[int]
    PARENTS_FIELD_NUMBER: _ClassVar[int]
    readable_id: str
    action: str
    timeout: str
    inputs: str
    parents: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, readable_id: _Optional[str] = ..., action: _Optional[str] = ..., timeout: _Optional[str] = ..., inputs: _Optional[str] = ..., parents: _Optional[_Iterable[str]] = ...) -> None: ...

class ListWorkflowsRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ScheduleWorkflowRequest(_message.Message):
    __slots__ = ("workflow_id", "schedules", "input")
    WORKFLOW_ID_FIELD_NUMBER: _ClassVar[int]
    SCHEDULES_FIELD_NUMBER: _ClassVar[int]
    INPUT_FIELD_NUMBER: _ClassVar[int]
    workflow_id: str
    schedules: _containers.RepeatedCompositeFieldContainer[_timestamp_pb2.Timestamp]
    input: str
    def __init__(self, workflow_id: _Optional[str] = ..., schedules: _Optional[_Iterable[_Union[_timestamp_pb2.Timestamp, _Mapping]]] = ..., input: _Optional[str] = ...) -> None: ...

class ListWorkflowsResponse(_message.Message):
    __slots__ = ("workflows",)
    WORKFLOWS_FIELD_NUMBER: _ClassVar[int]
    workflows: _containers.RepeatedCompositeFieldContainer[Workflow]
    def __init__(self, workflows: _Optional[_Iterable[_Union[Workflow, _Mapping]]] = ...) -> None: ...

class ListWorkflowsForEventRequest(_message.Message):
    __slots__ = ("event_key",)
    EVENT_KEY_FIELD_NUMBER: _ClassVar[int]
    event_key: str
    def __init__(self, event_key: _Optional[str] = ...) -> None: ...

class Workflow(_message.Message):
    __slots__ = ("id", "created_at", "updated_at", "tenant_id", "name", "description", "versions")
    ID_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    TENANT_ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    VERSIONS_FIELD_NUMBER: _ClassVar[int]
    id: str
    created_at: _timestamp_pb2.Timestamp
    updated_at: _timestamp_pb2.Timestamp
    tenant_id: str
    name: str
    description: _wrappers_pb2.StringValue
    versions: _containers.RepeatedCompositeFieldContainer[WorkflowVersion]
    def __init__(self, id: _Optional[str] = ..., created_at: _Optional[_Union[_timestamp_pb2.Timestamp, _Mapping]] = ..., updated_at: _Optional[_Union[_timestamp_pb2.Timestamp, _Mapping]] = ..., tenant_id: _Optional[str] = ..., name: _Optional[str] = ..., description: _Optional[_Union[_wrappers_pb2.StringValue, _Mapping]] = ..., versions: _Optional[_Iterable[_Union[WorkflowVersion, _Mapping]]] = ...) -> None: ...

class WorkflowVersion(_message.Message):
    __slots__ = ("id", "created_at", "updated_at", "version", "order", "workflow_id", "triggers", "jobs")
    ID_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    ORDER_FIELD_NUMBER: _ClassVar[int]
    WORKFLOW_ID_FIELD_NUMBER: _ClassVar[int]
    TRIGGERS_FIELD_NUMBER: _ClassVar[int]
    JOBS_FIELD_NUMBER: _ClassVar[int]
    id: str
    created_at: _timestamp_pb2.Timestamp
    updated_at: _timestamp_pb2.Timestamp
    version: str
    order: int
    workflow_id: str
    triggers: WorkflowTriggers
    jobs: _containers.RepeatedCompositeFieldContainer[Job]
    def __init__(self, id: _Optional[str] = ..., created_at: _Optional[_Union[_timestamp_pb2.Timestamp, _Mapping]] = ..., updated_at: _Optional[_Union[_timestamp_pb2.Timestamp, _Mapping]] = ..., version: _Optional[str] = ..., order: _Optional[int] = ..., workflow_id: _Optional[str] = ..., triggers: _Optional[_Union[WorkflowTriggers, _Mapping]] = ..., jobs: _Optional[_Iterable[_Union[Job, _Mapping]]] = ...) -> None: ...

class WorkflowTriggers(_message.Message):
    __slots__ = ("id", "created_at", "updated_at", "workflow_version_id", "tenant_id", "events", "crons")
    ID_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    WORKFLOW_VERSION_ID_FIELD_NUMBER: _ClassVar[int]
    TENANT_ID_FIELD_NUMBER: _ClassVar[int]
    EVENTS_FIELD_NUMBER: _ClassVar[int]
    CRONS_FIELD_NUMBER: _ClassVar[int]
    id: str
    created_at: _timestamp_pb2.Timestamp
    updated_at: _timestamp_pb2.Timestamp
    workflow_version_id: str
    tenant_id: str
    events: _containers.RepeatedCompositeFieldContainer[WorkflowTriggerEventRef]
    crons: _containers.RepeatedCompositeFieldContainer[WorkflowTriggerCronRef]
    def __init__(self, id: _Optional[str] = ..., created_at: _Optional[_Union[_timestamp_pb2.Timestamp, _Mapping]] = ..., updated_at: _Optional[_Union[_timestamp_pb2.Timestamp, _Mapping]] = ..., workflow_version_id: _Optional[str] = ..., tenant_id: _Optional[str] = ..., events: _Optional[_Iterable[_Union[WorkflowTriggerEventRef, _Mapping]]] = ..., crons: _Optional[_Iterable[_Union[WorkflowTriggerCronRef, _Mapping]]] = ...) -> None: ...

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

class Job(_message.Message):
    __slots__ = ("id", "created_at", "updated_at", "tenant_id", "workflow_version_id", "name", "description", "steps", "timeout")
    ID_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    TENANT_ID_FIELD_NUMBER: _ClassVar[int]
    WORKFLOW_VERSION_ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    STEPS_FIELD_NUMBER: _ClassVar[int]
    TIMEOUT_FIELD_NUMBER: _ClassVar[int]
    id: str
    created_at: _timestamp_pb2.Timestamp
    updated_at: _timestamp_pb2.Timestamp
    tenant_id: str
    workflow_version_id: str
    name: str
    description: _wrappers_pb2.StringValue
    steps: _containers.RepeatedCompositeFieldContainer[Step]
    timeout: _wrappers_pb2.StringValue
    def __init__(self, id: _Optional[str] = ..., created_at: _Optional[_Union[_timestamp_pb2.Timestamp, _Mapping]] = ..., updated_at: _Optional[_Union[_timestamp_pb2.Timestamp, _Mapping]] = ..., tenant_id: _Optional[str] = ..., workflow_version_id: _Optional[str] = ..., name: _Optional[str] = ..., description: _Optional[_Union[_wrappers_pb2.StringValue, _Mapping]] = ..., steps: _Optional[_Iterable[_Union[Step, _Mapping]]] = ..., timeout: _Optional[_Union[_wrappers_pb2.StringValue, _Mapping]] = ...) -> None: ...

class Step(_message.Message):
    __slots__ = ("id", "created_at", "updated_at", "readable_id", "tenant_id", "job_id", "action", "timeout", "parents", "children")
    ID_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    READABLE_ID_FIELD_NUMBER: _ClassVar[int]
    TENANT_ID_FIELD_NUMBER: _ClassVar[int]
    JOB_ID_FIELD_NUMBER: _ClassVar[int]
    ACTION_FIELD_NUMBER: _ClassVar[int]
    TIMEOUT_FIELD_NUMBER: _ClassVar[int]
    PARENTS_FIELD_NUMBER: _ClassVar[int]
    CHILDREN_FIELD_NUMBER: _ClassVar[int]
    id: str
    created_at: _timestamp_pb2.Timestamp
    updated_at: _timestamp_pb2.Timestamp
    readable_id: _wrappers_pb2.StringValue
    tenant_id: str
    job_id: str
    action: str
    timeout: _wrappers_pb2.StringValue
    parents: _containers.RepeatedScalarFieldContainer[str]
    children: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, id: _Optional[str] = ..., created_at: _Optional[_Union[_timestamp_pb2.Timestamp, _Mapping]] = ..., updated_at: _Optional[_Union[_timestamp_pb2.Timestamp, _Mapping]] = ..., readable_id: _Optional[_Union[_wrappers_pb2.StringValue, _Mapping]] = ..., tenant_id: _Optional[str] = ..., job_id: _Optional[str] = ..., action: _Optional[str] = ..., timeout: _Optional[_Union[_wrappers_pb2.StringValue, _Mapping]] = ..., parents: _Optional[_Iterable[str]] = ..., children: _Optional[_Iterable[str]] = ...) -> None: ...

class DeleteWorkflowRequest(_message.Message):
    __slots__ = ("workflow_id",)
    WORKFLOW_ID_FIELD_NUMBER: _ClassVar[int]
    workflow_id: str
    def __init__(self, workflow_id: _Optional[str] = ...) -> None: ...

class GetWorkflowByNameRequest(_message.Message):
    __slots__ = ("name",)
    NAME_FIELD_NUMBER: _ClassVar[int]
    name: str
    def __init__(self, name: _Optional[str] = ...) -> None: ...

class TriggerWorkflowRequest(_message.Message):
    __slots__ = ("name", "input")
    NAME_FIELD_NUMBER: _ClassVar[int]
    INPUT_FIELD_NUMBER: _ClassVar[int]
    name: str
    input: str
    def __init__(self, name: _Optional[str] = ..., input: _Optional[str] = ...) -> None: ...

class TriggerWorkflowResponse(_message.Message):
    __slots__ = ("workflow_run_id",)
    WORKFLOW_RUN_ID_FIELD_NUMBER: _ClassVar[int]
    workflow_run_id: str
    def __init__(self, workflow_run_id: _Optional[str] = ...) -> None: ...
