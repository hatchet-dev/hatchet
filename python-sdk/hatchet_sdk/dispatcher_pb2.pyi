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

class ActionEventType(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    STEP_EVENT_TYPE_UNKNOWN: _ClassVar[ActionEventType]
    STEP_EVENT_TYPE_STARTED: _ClassVar[ActionEventType]
    STEP_EVENT_TYPE_COMPLETED: _ClassVar[ActionEventType]
    STEP_EVENT_TYPE_FAILED: _ClassVar[ActionEventType]
START_STEP_RUN: ActionType
CANCEL_STEP_RUN: ActionType
STEP_EVENT_TYPE_UNKNOWN: ActionEventType
STEP_EVENT_TYPE_STARTED: ActionEventType
STEP_EVENT_TYPE_COMPLETED: ActionEventType
STEP_EVENT_TYPE_FAILED: ActionEventType

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
    __slots__ = ("tenantId", "jobId", "jobName", "jobRunId", "stepId", "stepRunId", "actionId", "actionType", "actionPayload")
    TENANTID_FIELD_NUMBER: _ClassVar[int]
    JOBID_FIELD_NUMBER: _ClassVar[int]
    JOBNAME_FIELD_NUMBER: _ClassVar[int]
    JOBRUNID_FIELD_NUMBER: _ClassVar[int]
    STEPID_FIELD_NUMBER: _ClassVar[int]
    STEPRUNID_FIELD_NUMBER: _ClassVar[int]
    ACTIONID_FIELD_NUMBER: _ClassVar[int]
    ACTIONTYPE_FIELD_NUMBER: _ClassVar[int]
    ACTIONPAYLOAD_FIELD_NUMBER: _ClassVar[int]
    tenantId: str
    jobId: str
    jobName: str
    jobRunId: str
    stepId: str
    stepRunId: str
    actionId: str
    actionType: ActionType
    actionPayload: str
    def __init__(self, tenantId: _Optional[str] = ..., jobId: _Optional[str] = ..., jobName: _Optional[str] = ..., jobRunId: _Optional[str] = ..., stepId: _Optional[str] = ..., stepRunId: _Optional[str] = ..., actionId: _Optional[str] = ..., actionType: _Optional[_Union[ActionType, str]] = ..., actionPayload: _Optional[str] = ...) -> None: ...

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

class ActionEvent(_message.Message):
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
    eventType: ActionEventType
    eventPayload: str
    def __init__(self, workerId: _Optional[str] = ..., jobId: _Optional[str] = ..., jobRunId: _Optional[str] = ..., stepId: _Optional[str] = ..., stepRunId: _Optional[str] = ..., actionId: _Optional[str] = ..., eventTimestamp: _Optional[_Union[_timestamp_pb2.Timestamp, _Mapping]] = ..., eventType: _Optional[_Union[ActionEventType, str]] = ..., eventPayload: _Optional[str] = ...) -> None: ...

class ActionEventResponse(_message.Message):
    __slots__ = ("tenantId", "workerId")
    TENANTID_FIELD_NUMBER: _ClassVar[int]
    WORKERID_FIELD_NUMBER: _ClassVar[int]
    tenantId: str
    workerId: str
    def __init__(self, tenantId: _Optional[str] = ..., workerId: _Optional[str] = ...) -> None: ...
