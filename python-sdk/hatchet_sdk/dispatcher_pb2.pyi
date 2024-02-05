from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from typing import ClassVar as _ClassVar, Iterable as _Iterable, Mapping as _Mapping, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ActionType(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    START_STEP_RUN: _ClassVar[ActionType]
    CANCEL_STEP_RUN: _ClassVar[ActionType]
    START_GET_GROUP_KEY: _ClassVar[ActionType]

class GroupKeyActionEventType(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    GROUP_KEY_EVENT_TYPE_UNKNOWN: _ClassVar[GroupKeyActionEventType]
    GROUP_KEY_EVENT_TYPE_STARTED: _ClassVar[GroupKeyActionEventType]
    GROUP_KEY_EVENT_TYPE_COMPLETED: _ClassVar[GroupKeyActionEventType]
    GROUP_KEY_EVENT_TYPE_FAILED: _ClassVar[GroupKeyActionEventType]

class StepActionEventType(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    STEP_EVENT_TYPE_UNKNOWN: _ClassVar[StepActionEventType]
    STEP_EVENT_TYPE_STARTED: _ClassVar[StepActionEventType]
    STEP_EVENT_TYPE_COMPLETED: _ClassVar[StepActionEventType]
    STEP_EVENT_TYPE_FAILED: _ClassVar[StepActionEventType]

class ResourceType(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    RESOURCE_TYPE_UNKNOWN: _ClassVar[ResourceType]
    RESOURCE_TYPE_STEP_RUN: _ClassVar[ResourceType]
    RESOURCE_TYPE_WORKFLOW_RUN: _ClassVar[ResourceType]

class ResourceEventType(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    RESOURCE_EVENT_TYPE_UNKNOWN: _ClassVar[ResourceEventType]
    RESOURCE_EVENT_TYPE_STARTED: _ClassVar[ResourceEventType]
    RESOURCE_EVENT_TYPE_COMPLETED: _ClassVar[ResourceEventType]
    RESOURCE_EVENT_TYPE_FAILED: _ClassVar[ResourceEventType]
    RESOURCE_EVENT_TYPE_CANCELLED: _ClassVar[ResourceEventType]
    RESOURCE_EVENT_TYPE_TIMED_OUT: _ClassVar[ResourceEventType]
START_STEP_RUN: ActionType
CANCEL_STEP_RUN: ActionType
START_GET_GROUP_KEY: ActionType
GROUP_KEY_EVENT_TYPE_UNKNOWN: GroupKeyActionEventType
GROUP_KEY_EVENT_TYPE_STARTED: GroupKeyActionEventType
GROUP_KEY_EVENT_TYPE_COMPLETED: GroupKeyActionEventType
GROUP_KEY_EVENT_TYPE_FAILED: GroupKeyActionEventType
STEP_EVENT_TYPE_UNKNOWN: StepActionEventType
STEP_EVENT_TYPE_STARTED: StepActionEventType
STEP_EVENT_TYPE_COMPLETED: StepActionEventType
STEP_EVENT_TYPE_FAILED: StepActionEventType
RESOURCE_TYPE_UNKNOWN: ResourceType
RESOURCE_TYPE_STEP_RUN: ResourceType
RESOURCE_TYPE_WORKFLOW_RUN: ResourceType
RESOURCE_EVENT_TYPE_UNKNOWN: ResourceEventType
RESOURCE_EVENT_TYPE_STARTED: ResourceEventType
RESOURCE_EVENT_TYPE_COMPLETED: ResourceEventType
RESOURCE_EVENT_TYPE_FAILED: ResourceEventType
RESOURCE_EVENT_TYPE_CANCELLED: ResourceEventType
RESOURCE_EVENT_TYPE_TIMED_OUT: ResourceEventType

class WorkerRegisterRequest(_message.Message):
    __slots__ = ("workerName", "actions", "services")
    WORKERNAME_FIELD_NUMBER: _ClassVar[int]
    ACTIONS_FIELD_NUMBER: _ClassVar[int]
    SERVICES_FIELD_NUMBER: _ClassVar[int]
    workerName: str
    actions: _containers.RepeatedScalarFieldContainer[str]
    services: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, workerName: _Optional[str] = ..., actions: _Optional[_Iterable[str]] = ..., services: _Optional[_Iterable[str]] = ...) -> None: ...

class WorkerRegisterResponse(_message.Message):
    __slots__ = ("tenantId", "workerId", "workerName")
    TENANTID_FIELD_NUMBER: _ClassVar[int]
    WORKERID_FIELD_NUMBER: _ClassVar[int]
    WORKERNAME_FIELD_NUMBER: _ClassVar[int]
    tenantId: str
    workerId: str
    workerName: str
    def __init__(self, tenantId: _Optional[str] = ..., workerId: _Optional[str] = ..., workerName: _Optional[str] = ...) -> None: ...

class AssignedAction(_message.Message):
    __slots__ = ("tenantId", "workflowRunId", "getGroupKeyRunId", "jobId", "jobName", "jobRunId", "stepId", "stepRunId", "actionId", "actionType", "actionPayload")
    TENANTID_FIELD_NUMBER: _ClassVar[int]
    WORKFLOWRUNID_FIELD_NUMBER: _ClassVar[int]
    GETGROUPKEYRUNID_FIELD_NUMBER: _ClassVar[int]
    JOBID_FIELD_NUMBER: _ClassVar[int]
    JOBNAME_FIELD_NUMBER: _ClassVar[int]
    JOBRUNID_FIELD_NUMBER: _ClassVar[int]
    STEPID_FIELD_NUMBER: _ClassVar[int]
    STEPRUNID_FIELD_NUMBER: _ClassVar[int]
    ACTIONID_FIELD_NUMBER: _ClassVar[int]
    ACTIONTYPE_FIELD_NUMBER: _ClassVar[int]
    ACTIONPAYLOAD_FIELD_NUMBER: _ClassVar[int]
    tenantId: str
    workflowRunId: str
    getGroupKeyRunId: str
    jobId: str
    jobName: str
    jobRunId: str
    stepId: str
    stepRunId: str
    actionId: str
    actionType: ActionType
    actionPayload: str
    def __init__(self, tenantId: _Optional[str] = ..., workflowRunId: _Optional[str] = ..., getGroupKeyRunId: _Optional[str] = ..., jobId: _Optional[str] = ..., jobName: _Optional[str] = ..., jobRunId: _Optional[str] = ..., stepId: _Optional[str] = ..., stepRunId: _Optional[str] = ..., actionId: _Optional[str] = ..., actionType: _Optional[_Union[ActionType, str]] = ..., actionPayload: _Optional[str] = ...) -> None: ...

class WorkerListenRequest(_message.Message):
    __slots__ = ("workerId",)
    WORKERID_FIELD_NUMBER: _ClassVar[int]
    workerId: str
    def __init__(self, workerId: _Optional[str] = ...) -> None: ...

class WorkerUnsubscribeRequest(_message.Message):
    __slots__ = ("workerId",)
    WORKERID_FIELD_NUMBER: _ClassVar[int]
    workerId: str
    def __init__(self, workerId: _Optional[str] = ...) -> None: ...

class WorkerUnsubscribeResponse(_message.Message):
    __slots__ = ("tenantId", "workerId")
    TENANTID_FIELD_NUMBER: _ClassVar[int]
    WORKERID_FIELD_NUMBER: _ClassVar[int]
    tenantId: str
    workerId: str
    def __init__(self, tenantId: _Optional[str] = ..., workerId: _Optional[str] = ...) -> None: ...

class GroupKeyActionEvent(_message.Message):
    __slots__ = ("workerId", "workflowRunId", "getGroupKeyRunId", "actionId", "eventTimestamp", "eventType", "eventPayload")
    WORKERID_FIELD_NUMBER: _ClassVar[int]
    WORKFLOWRUNID_FIELD_NUMBER: _ClassVar[int]
    GETGROUPKEYRUNID_FIELD_NUMBER: _ClassVar[int]
    ACTIONID_FIELD_NUMBER: _ClassVar[int]
    EVENTTIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    EVENTTYPE_FIELD_NUMBER: _ClassVar[int]
    EVENTPAYLOAD_FIELD_NUMBER: _ClassVar[int]
    workerId: str
    workflowRunId: str
    getGroupKeyRunId: str
    actionId: str
    eventTimestamp: _timestamp_pb2.Timestamp
    eventType: GroupKeyActionEventType
    eventPayload: str
    def __init__(self, workerId: _Optional[str] = ..., workflowRunId: _Optional[str] = ..., getGroupKeyRunId: _Optional[str] = ..., actionId: _Optional[str] = ..., eventTimestamp: _Optional[_Union[_timestamp_pb2.Timestamp, _Mapping]] = ..., eventType: _Optional[_Union[GroupKeyActionEventType, str]] = ..., eventPayload: _Optional[str] = ...) -> None: ...

class StepActionEvent(_message.Message):
    __slots__ = ("workerId", "jobId", "jobRunId", "stepId", "stepRunId", "actionId", "eventTimestamp", "eventType", "eventPayload")
    WORKERID_FIELD_NUMBER: _ClassVar[int]
    JOBID_FIELD_NUMBER: _ClassVar[int]
    JOBRUNID_FIELD_NUMBER: _ClassVar[int]
    STEPID_FIELD_NUMBER: _ClassVar[int]
    STEPRUNID_FIELD_NUMBER: _ClassVar[int]
    ACTIONID_FIELD_NUMBER: _ClassVar[int]
    EVENTTIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    EVENTTYPE_FIELD_NUMBER: _ClassVar[int]
    EVENTPAYLOAD_FIELD_NUMBER: _ClassVar[int]
    workerId: str
    jobId: str
    jobRunId: str
    stepId: str
    stepRunId: str
    actionId: str
    eventTimestamp: _timestamp_pb2.Timestamp
    eventType: StepActionEventType
    eventPayload: str
    def __init__(self, workerId: _Optional[str] = ..., jobId: _Optional[str] = ..., jobRunId: _Optional[str] = ..., stepId: _Optional[str] = ..., stepRunId: _Optional[str] = ..., actionId: _Optional[str] = ..., eventTimestamp: _Optional[_Union[_timestamp_pb2.Timestamp, _Mapping]] = ..., eventType: _Optional[_Union[StepActionEventType, str]] = ..., eventPayload: _Optional[str] = ...) -> None: ...

class ActionEventResponse(_message.Message):
    __slots__ = ("tenantId", "workerId")
    TENANTID_FIELD_NUMBER: _ClassVar[int]
    WORKERID_FIELD_NUMBER: _ClassVar[int]
    tenantId: str
    workerId: str
    def __init__(self, tenantId: _Optional[str] = ..., workerId: _Optional[str] = ...) -> None: ...

class SubscribeToWorkflowEventsRequest(_message.Message):
    __slots__ = ("workflowRunId",)
    WORKFLOWRUNID_FIELD_NUMBER: _ClassVar[int]
    workflowRunId: str
    def __init__(self, workflowRunId: _Optional[str] = ...) -> None: ...

class WorkflowEvent(_message.Message):
    __slots__ = ("workflowRunId", "resourceType", "eventType", "resourceId", "eventTimestamp", "eventPayload")
    WORKFLOWRUNID_FIELD_NUMBER: _ClassVar[int]
    RESOURCETYPE_FIELD_NUMBER: _ClassVar[int]
    EVENTTYPE_FIELD_NUMBER: _ClassVar[int]
    RESOURCEID_FIELD_NUMBER: _ClassVar[int]
    EVENTTIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    EVENTPAYLOAD_FIELD_NUMBER: _ClassVar[int]
    workflowRunId: str
    resourceType: ResourceType
    eventType: ResourceEventType
    resourceId: str
    eventTimestamp: _timestamp_pb2.Timestamp
    eventPayload: str
    def __init__(self, workflowRunId: _Optional[str] = ..., resourceType: _Optional[_Union[ResourceType, str]] = ..., eventType: _Optional[_Union[ResourceEventType, str]] = ..., resourceId: _Optional[str] = ..., eventTimestamp: _Optional[_Union[_timestamp_pb2.Timestamp, _Mapping]] = ..., eventPayload: _Optional[str] = ...) -> None: ...
