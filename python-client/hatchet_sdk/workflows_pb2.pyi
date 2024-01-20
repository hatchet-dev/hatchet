from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf import wrappers_pb2 as _wrappers_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from typing import ClassVar as _ClassVar, Iterable as _Iterable, Mapping as _Mapping, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class PutWorkflowRequest(_message.Message):
    __slots__ = ("tenant_id", "opts")
    TENANT_ID_FIELD_NUMBER: _ClassVar[int]
    OPTS_FIELD_NUMBER: _ClassVar[int]
    tenant_id: str
    opts: CreateWorkflowVersionOpts
    def __init__(self, tenant_id: _Optional[str] = ..., opts: _Optional[_Union[CreateWorkflowVersionOpts, _Mapping]] = ...) -> None: ...

class CreateWorkflowVersionOpts(_message.Message):
    __slots__ = ("name", "description", "version", "event_triggers", "cron_triggers", "scheduled_triggers", "jobs")
    NAME_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    EVENT_TRIGGERS_FIELD_NUMBER: _ClassVar[int]
    CRON_TRIGGERS_FIELD_NUMBER: _ClassVar[int]
    SCHEDULED_TRIGGERS_FIELD_NUMBER: _ClassVar[int]
    JOBS_FIELD_NUMBER: _ClassVar[int]
    name: str
    description: str
    version: str
    event_triggers: _containers.RepeatedScalarFieldContainer[str]
    cron_triggers: _containers.RepeatedScalarFieldContainer[str]
    scheduled_triggers: _containers.RepeatedCompositeFieldContainer[_timestamp_pb2.Timestamp]
    jobs: _containers.RepeatedCompositeFieldContainer[CreateWorkflowJobOpts]
    def __init__(self, name: _Optional[str] = ..., description: _Optional[str] = ..., version: _Optional[str] = ..., event_triggers: _Optional[_Iterable[str]] = ..., cron_triggers: _Optional[_Iterable[str]] = ..., scheduled_triggers: _Optional[_Iterable[_Union[_timestamp_pb2.Timestamp, _Mapping]]] = ..., jobs: _Optional[_Iterable[_Union[CreateWorkflowJobOpts, _Mapping]]] = ...) -> None: ...

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
    __slots__ = ("tenant_id",)
    TENANT_ID_FIELD_NUMBER: _ClassVar[int]
    tenant_id: str
    def __init__(self, tenant_id: _Optional[str] = ...) -> None: ...

class ScheduleWorkflowRequest(_message.Message):
    __slots__ = ("tenant_id", "workflow_id", "schedules", "input")
    TENANT_ID_FIELD_NUMBER: _ClassVar[int]
    WORKFLOW_ID_FIELD_NUMBER: _ClassVar[int]
    SCHEDULES_FIELD_NUMBER: _ClassVar[int]
    INPUT_FIELD_NUMBER: _ClassVar[int]
    tenant_id: str
    workflow_id: str
    schedules: _containers.RepeatedCompositeFieldContainer[_timestamp_pb2.Timestamp]
    input: str
    def __init__(self, tenant_id: _Optional[str] = ..., workflow_id: _Optional[str] = ..., schedules: _Optional[_Iterable[_Union[_timestamp_pb2.Timestamp, _Mapping]]] = ..., input: _Optional[str] = ...) -> None: ...

class ListWorkflowsResponse(_message.Message):
    __slots__ = ("workflows",)
    WORKFLOWS_FIELD_NUMBER: _ClassVar[int]
    workflows: _containers.RepeatedCompositeFieldContainer[Workflow]
    def __init__(self, workflows: _Optional[_Iterable[_Union[Workflow, _Mapping]]] = ...) -> None: ...

class ListWorkflowsForEventRequest(_message.Message):
    __slots__ = ("tenant_id", "event_key")
    TENANT_ID_FIELD_NUMBER: _ClassVar[int]
    EVENT_KEY_FIELD_NUMBER: _ClassVar[int]
    tenant_id: str
    event_key: str
    def __init__(self, tenant_id: _Optional[str] = ..., event_key: _Optional[str] = ...) -> None: ...

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
    __slots__ = ("tenant_id", "workflow_id")
    TENANT_ID_FIELD_NUMBER: _ClassVar[int]
    WORKFLOW_ID_FIELD_NUMBER: _ClassVar[int]
    tenant_id: str
    workflow_id: str
    def __init__(self, tenant_id: _Optional[str] = ..., workflow_id: _Optional[str] = ...) -> None: ...

class GetWorkflowByNameRequest(_message.Message):
    __slots__ = ("tenant_id", "name")
    TENANT_ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    tenant_id: str
    name: str
    def __init__(self, tenant_id: _Optional[str] = ..., name: _Optional[str] = ...) -> None: ...
