from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from typing import ClassVar as _ClassVar, Iterable as _Iterable, Mapping as _Mapping, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class SDKS(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    UNKNOWN: _ClassVar[SDKS]
    GO: _ClassVar[SDKS]
    PYTHON: _ClassVar[SDKS]
    TYPESCRIPT: _ClassVar[SDKS]

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
    STEP_EVENT_TYPE_ACKNOWLEDGED: _ClassVar[StepActionEventType]

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
    RESOURCE_EVENT_TYPE_STREAM: _ClassVar[ResourceEventType]

class WorkflowRunEventType(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    WORKFLOW_RUN_EVENT_TYPE_FINISHED: _ClassVar[WorkflowRunEventType]
UNKNOWN: SDKS
GO: SDKS
PYTHON: SDKS
TYPESCRIPT: SDKS
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
STEP_EVENT_TYPE_ACKNOWLEDGED: StepActionEventType
RESOURCE_TYPE_UNKNOWN: ResourceType
RESOURCE_TYPE_STEP_RUN: ResourceType
RESOURCE_TYPE_WORKFLOW_RUN: ResourceType
RESOURCE_EVENT_TYPE_UNKNOWN: ResourceEventType
RESOURCE_EVENT_TYPE_STARTED: ResourceEventType
RESOURCE_EVENT_TYPE_COMPLETED: ResourceEventType
RESOURCE_EVENT_TYPE_FAILED: ResourceEventType
RESOURCE_EVENT_TYPE_CANCELLED: ResourceEventType
RESOURCE_EVENT_TYPE_TIMED_OUT: ResourceEventType
RESOURCE_EVENT_TYPE_STREAM: ResourceEventType
WORKFLOW_RUN_EVENT_TYPE_FINISHED: WorkflowRunEventType

class WorkerLabels(_message.Message):
    __slots__ = ("strValue", "intValue")
    STRVALUE_FIELD_NUMBER: _ClassVar[int]
    INTVALUE_FIELD_NUMBER: _ClassVar[int]
    strValue: str
    intValue: int
    def __init__(self, strValue: _Optional[str] = ..., intValue: _Optional[int] = ...) -> None: ...

class RuntimeInfo(_message.Message):
    __slots__ = ("sdkVersion", "language", "languageVersion", "os", "extra")
    SDKVERSION_FIELD_NUMBER: _ClassVar[int]
    LANGUAGE_FIELD_NUMBER: _ClassVar[int]
    LANGUAGEVERSION_FIELD_NUMBER: _ClassVar[int]
    OS_FIELD_NUMBER: _ClassVar[int]
    EXTRA_FIELD_NUMBER: _ClassVar[int]
    sdkVersion: str
    language: SDKS
    languageVersion: str
    os: str
    extra: str
    def __init__(self, sdkVersion: _Optional[str] = ..., language: _Optional[_Union[SDKS, str]] = ..., languageVersion: _Optional[str] = ..., os: _Optional[str] = ..., extra: _Optional[str] = ...) -> None: ...

class WorkerRegisterRequest(_message.Message):
    __slots__ = ("workerName", "actions", "services", "maxRuns", "labels", "webhookId", "runtimeInfo")
    class LabelsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: WorkerLabels
        def __init__(self, key: _Optional[str] = ..., value: _Optional[_Union[WorkerLabels, _Mapping]] = ...) -> None: ...
    WORKERNAME_FIELD_NUMBER: _ClassVar[int]
    ACTIONS_FIELD_NUMBER: _ClassVar[int]
    SERVICES_FIELD_NUMBER: _ClassVar[int]
    MAXRUNS_FIELD_NUMBER: _ClassVar[int]
    LABELS_FIELD_NUMBER: _ClassVar[int]
    WEBHOOKID_FIELD_NUMBER: _ClassVar[int]
    RUNTIMEINFO_FIELD_NUMBER: _ClassVar[int]
    workerName: str
    actions: _containers.RepeatedScalarFieldContainer[str]
    services: _containers.RepeatedScalarFieldContainer[str]
    maxRuns: int
    labels: _containers.MessageMap[str, WorkerLabels]
    webhookId: str
    runtimeInfo: RuntimeInfo
    def __init__(self, workerName: _Optional[str] = ..., actions: _Optional[_Iterable[str]] = ..., services: _Optional[_Iterable[str]] = ..., maxRuns: _Optional[int] = ..., labels: _Optional[_Mapping[str, WorkerLabels]] = ..., webhookId: _Optional[str] = ..., runtimeInfo: _Optional[_Union[RuntimeInfo, _Mapping]] = ...) -> None: ...

class WorkerRegisterResponse(_message.Message):
    __slots__ = ("tenantId", "workerId", "workerName")
    TENANTID_FIELD_NUMBER: _ClassVar[int]
    WORKERID_FIELD_NUMBER: _ClassVar[int]
    WORKERNAME_FIELD_NUMBER: _ClassVar[int]
    tenantId: str
    workerId: str
    workerName: str
    def __init__(self, tenantId: _Optional[str] = ..., workerId: _Optional[str] = ..., workerName: _Optional[str] = ...) -> None: ...

class UpsertWorkerLabelsRequest(_message.Message):
    __slots__ = ("workerId", "labels")
    class LabelsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: WorkerLabels
        def __init__(self, key: _Optional[str] = ..., value: _Optional[_Union[WorkerLabels, _Mapping]] = ...) -> None: ...
    WORKERID_FIELD_NUMBER: _ClassVar[int]
    LABELS_FIELD_NUMBER: _ClassVar[int]
    workerId: str
    labels: _containers.MessageMap[str, WorkerLabels]
    def __init__(self, workerId: _Optional[str] = ..., labels: _Optional[_Mapping[str, WorkerLabels]] = ...) -> None: ...

class UpsertWorkerLabelsResponse(_message.Message):
    __slots__ = ("tenantId", "workerId")
    TENANTID_FIELD_NUMBER: _ClassVar[int]
    WORKERID_FIELD_NUMBER: _ClassVar[int]
    tenantId: str
    workerId: str
    def __init__(self, tenantId: _Optional[str] = ..., workerId: _Optional[str] = ...) -> None: ...

class AssignedAction(_message.Message):
    __slots__ = ("tenantId", "workflowRunId", "getGroupKeyRunId", "jobId", "jobName", "jobRunId", "stepId", "stepRunId", "actionId", "actionType", "actionPayload", "stepName", "retryCount", "additional_metadata", "child_workflow_index", "child_workflow_key", "parent_workflow_run_id", "priority", "workflowId", "workflowVersionId")
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
    STEPNAME_FIELD_NUMBER: _ClassVar[int]
    RETRYCOUNT_FIELD_NUMBER: _ClassVar[int]
    ADDITIONAL_METADATA_FIELD_NUMBER: _ClassVar[int]
    CHILD_WORKFLOW_INDEX_FIELD_NUMBER: _ClassVar[int]
    CHILD_WORKFLOW_KEY_FIELD_NUMBER: _ClassVar[int]
    PARENT_WORKFLOW_RUN_ID_FIELD_NUMBER: _ClassVar[int]
    PRIORITY_FIELD_NUMBER: _ClassVar[int]
    WORKFLOWID_FIELD_NUMBER: _ClassVar[int]
    WORKFLOWVERSIONID_FIELD_NUMBER: _ClassVar[int]
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
    stepName: str
    retryCount: int
    additional_metadata: str
    child_workflow_index: int
    child_workflow_key: str
    parent_workflow_run_id: str
    priority: int
    workflowId: str
    workflowVersionId: str
    def __init__(self, tenantId: _Optional[str] = ..., workflowRunId: _Optional[str] = ..., getGroupKeyRunId: _Optional[str] = ..., jobId: _Optional[str] = ..., jobName: _Optional[str] = ..., jobRunId: _Optional[str] = ..., stepId: _Optional[str] = ..., stepRunId: _Optional[str] = ..., actionId: _Optional[str] = ..., actionType: _Optional[_Union[ActionType, str]] = ..., actionPayload: _Optional[str] = ..., stepName: _Optional[str] = ..., retryCount: _Optional[int] = ..., additional_metadata: _Optional[str] = ..., child_workflow_index: _Optional[int] = ..., child_workflow_key: _Optional[str] = ..., parent_workflow_run_id: _Optional[str] = ..., priority: _Optional[int] = ..., workflowId: _Optional[str] = ..., workflowVersionId: _Optional[str] = ...) -> None: ...

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
    __slots__ = ("workerId", "jobId", "jobRunId", "stepId", "stepRunId", "actionId", "eventTimestamp", "eventType", "eventPayload", "retryCount", "shouldNotRetry")
    WORKERID_FIELD_NUMBER: _ClassVar[int]
    JOBID_FIELD_NUMBER: _ClassVar[int]
    JOBRUNID_FIELD_NUMBER: _ClassVar[int]
    STEPID_FIELD_NUMBER: _ClassVar[int]
    STEPRUNID_FIELD_NUMBER: _ClassVar[int]
    ACTIONID_FIELD_NUMBER: _ClassVar[int]
    EVENTTIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    EVENTTYPE_FIELD_NUMBER: _ClassVar[int]
    EVENTPAYLOAD_FIELD_NUMBER: _ClassVar[int]
    RETRYCOUNT_FIELD_NUMBER: _ClassVar[int]
    SHOULDNOTRETRY_FIELD_NUMBER: _ClassVar[int]
    workerId: str
    jobId: str
    jobRunId: str
    stepId: str
    stepRunId: str
    actionId: str
    eventTimestamp: _timestamp_pb2.Timestamp
    eventType: StepActionEventType
    eventPayload: str
    retryCount: int
    shouldNotRetry: bool
    def __init__(self, workerId: _Optional[str] = ..., jobId: _Optional[str] = ..., jobRunId: _Optional[str] = ..., stepId: _Optional[str] = ..., stepRunId: _Optional[str] = ..., actionId: _Optional[str] = ..., eventTimestamp: _Optional[_Union[_timestamp_pb2.Timestamp, _Mapping]] = ..., eventType: _Optional[_Union[StepActionEventType, str]] = ..., eventPayload: _Optional[str] = ..., retryCount: _Optional[int] = ..., shouldNotRetry: bool = ...) -> None: ...

class ActionEventResponse(_message.Message):
    __slots__ = ("tenantId", "workerId")
    TENANTID_FIELD_NUMBER: _ClassVar[int]
    WORKERID_FIELD_NUMBER: _ClassVar[int]
    tenantId: str
    workerId: str
    def __init__(self, tenantId: _Optional[str] = ..., workerId: _Optional[str] = ...) -> None: ...

class SubscribeToWorkflowEventsRequest(_message.Message):
    __slots__ = ("workflowRunId", "additionalMetaKey", "additionalMetaValue")
    WORKFLOWRUNID_FIELD_NUMBER: _ClassVar[int]
    ADDITIONALMETAKEY_FIELD_NUMBER: _ClassVar[int]
    ADDITIONALMETAVALUE_FIELD_NUMBER: _ClassVar[int]
    workflowRunId: str
    additionalMetaKey: str
    additionalMetaValue: str
    def __init__(self, workflowRunId: _Optional[str] = ..., additionalMetaKey: _Optional[str] = ..., additionalMetaValue: _Optional[str] = ...) -> None: ...

class SubscribeToWorkflowRunsRequest(_message.Message):
    __slots__ = ("workflowRunId",)
    WORKFLOWRUNID_FIELD_NUMBER: _ClassVar[int]
    workflowRunId: str
    def __init__(self, workflowRunId: _Optional[str] = ...) -> None: ...

class WorkflowEvent(_message.Message):
    __slots__ = ("workflowRunId", "resourceType", "eventType", "resourceId", "eventTimestamp", "eventPayload", "hangup", "stepRetries", "retryCount", "eventIndex")
    WORKFLOWRUNID_FIELD_NUMBER: _ClassVar[int]
    RESOURCETYPE_FIELD_NUMBER: _ClassVar[int]
    EVENTTYPE_FIELD_NUMBER: _ClassVar[int]
    RESOURCEID_FIELD_NUMBER: _ClassVar[int]
    EVENTTIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    EVENTPAYLOAD_FIELD_NUMBER: _ClassVar[int]
    HANGUP_FIELD_NUMBER: _ClassVar[int]
    STEPRETRIES_FIELD_NUMBER: _ClassVar[int]
    RETRYCOUNT_FIELD_NUMBER: _ClassVar[int]
    EVENTINDEX_FIELD_NUMBER: _ClassVar[int]
    workflowRunId: str
    resourceType: ResourceType
    eventType: ResourceEventType
    resourceId: str
    eventTimestamp: _timestamp_pb2.Timestamp
    eventPayload: str
    hangup: bool
    stepRetries: int
    retryCount: int
    eventIndex: int
    def __init__(self, workflowRunId: _Optional[str] = ..., resourceType: _Optional[_Union[ResourceType, str]] = ..., eventType: _Optional[_Union[ResourceEventType, str]] = ..., resourceId: _Optional[str] = ..., eventTimestamp: _Optional[_Union[_timestamp_pb2.Timestamp, _Mapping]] = ..., eventPayload: _Optional[str] = ..., hangup: bool = ..., stepRetries: _Optional[int] = ..., retryCount: _Optional[int] = ..., eventIndex: _Optional[int] = ...) -> None: ...

class WorkflowRunEvent(_message.Message):
    __slots__ = ("workflowRunId", "eventType", "eventTimestamp", "results")
    WORKFLOWRUNID_FIELD_NUMBER: _ClassVar[int]
    EVENTTYPE_FIELD_NUMBER: _ClassVar[int]
    EVENTTIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    RESULTS_FIELD_NUMBER: _ClassVar[int]
    workflowRunId: str
    eventType: WorkflowRunEventType
    eventTimestamp: _timestamp_pb2.Timestamp
    results: _containers.RepeatedCompositeFieldContainer[StepRunResult]
    def __init__(self, workflowRunId: _Optional[str] = ..., eventType: _Optional[_Union[WorkflowRunEventType, str]] = ..., eventTimestamp: _Optional[_Union[_timestamp_pb2.Timestamp, _Mapping]] = ..., results: _Optional[_Iterable[_Union[StepRunResult, _Mapping]]] = ...) -> None: ...

class StepRunResult(_message.Message):
    __slots__ = ("stepRunId", "stepReadableId", "jobRunId", "error", "output")
    STEPRUNID_FIELD_NUMBER: _ClassVar[int]
    STEPREADABLEID_FIELD_NUMBER: _ClassVar[int]
    JOBRUNID_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    OUTPUT_FIELD_NUMBER: _ClassVar[int]
    stepRunId: str
    stepReadableId: str
    jobRunId: str
    error: str
    output: str
    def __init__(self, stepRunId: _Optional[str] = ..., stepReadableId: _Optional[str] = ..., jobRunId: _Optional[str] = ..., error: _Optional[str] = ..., output: _Optional[str] = ...) -> None: ...

class OverridesData(_message.Message):
    __slots__ = ("stepRunId", "path", "value", "callerFilename")
    STEPRUNID_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    VALUE_FIELD_NUMBER: _ClassVar[int]
    CALLERFILENAME_FIELD_NUMBER: _ClassVar[int]
    stepRunId: str
    path: str
    value: str
    callerFilename: str
    def __init__(self, stepRunId: _Optional[str] = ..., path: _Optional[str] = ..., value: _Optional[str] = ..., callerFilename: _Optional[str] = ...) -> None: ...

class OverridesDataResponse(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class HeartbeatRequest(_message.Message):
    __slots__ = ("workerId", "heartbeatAt")
    WORKERID_FIELD_NUMBER: _ClassVar[int]
    HEARTBEATAT_FIELD_NUMBER: _ClassVar[int]
    workerId: str
    heartbeatAt: _timestamp_pb2.Timestamp
    def __init__(self, workerId: _Optional[str] = ..., heartbeatAt: _Optional[_Union[_timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class HeartbeatResponse(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class RefreshTimeoutRequest(_message.Message):
    __slots__ = ("stepRunId", "incrementTimeoutBy")
    STEPRUNID_FIELD_NUMBER: _ClassVar[int]
    INCREMENTTIMEOUTBY_FIELD_NUMBER: _ClassVar[int]
    stepRunId: str
    incrementTimeoutBy: str
    def __init__(self, stepRunId: _Optional[str] = ..., incrementTimeoutBy: _Optional[str] = ...) -> None: ...

class RefreshTimeoutResponse(_message.Message):
    __slots__ = ("timeoutAt",)
    TIMEOUTAT_FIELD_NUMBER: _ClassVar[int]
    timeoutAt: _timestamp_pb2.Timestamp
    def __init__(self, timeoutAt: _Optional[_Union[_timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class ReleaseSlotRequest(_message.Message):
    __slots__ = ("stepRunId",)
    STEPRUNID_FIELD_NUMBER: _ClassVar[int]
    stepRunId: str
    def __init__(self, stepRunId: _Optional[str] = ...) -> None: ...

class ReleaseSlotResponse(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...
