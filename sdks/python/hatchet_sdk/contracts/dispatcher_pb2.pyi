import datetime
from collections.abc import Iterable as _Iterable
from collections.abc import Mapping as _Mapping
from typing import ClassVar as _ClassVar
from typing import Optional as _Optional
from typing import Union as _Union

from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper

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
    __slots__ = ("str_value", "int_value")
    STR_VALUE_FIELD_NUMBER: _ClassVar[int]
    INT_VALUE_FIELD_NUMBER: _ClassVar[int]
    str_value: str
    int_value: int
    def __init__(self, str_value: _Optional[str] = ..., int_value: _Optional[int] = ...) -> None: ...

class RuntimeInfo(_message.Message):
    __slots__ = ("sdk_version", "language", "language_version", "os", "extra")
    SDK_VERSION_FIELD_NUMBER: _ClassVar[int]
    LANGUAGE_FIELD_NUMBER: _ClassVar[int]
    LANGUAGE_VERSION_FIELD_NUMBER: _ClassVar[int]
    OS_FIELD_NUMBER: _ClassVar[int]
    EXTRA_FIELD_NUMBER: _ClassVar[int]
    sdk_version: str
    language: SDKS
    language_version: str
    os: str
    extra: str
    def __init__(self, sdk_version: _Optional[str] = ..., language: _Optional[_Union[SDKS, str]] = ..., language_version: _Optional[str] = ..., os: _Optional[str] = ..., extra: _Optional[str] = ...) -> None: ...

class WorkerRegisterRequest(_message.Message):
    __slots__ = ("worker_name", "actions", "services", "slots", "labels", "webhook_id", "runtime_info")
    class LabelsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: WorkerLabels
        def __init__(self, key: _Optional[str] = ..., value: _Optional[_Union[WorkerLabels, _Mapping]] = ...) -> None: ...
    WORKER_NAME_FIELD_NUMBER: _ClassVar[int]
    ACTIONS_FIELD_NUMBER: _ClassVar[int]
    SERVICES_FIELD_NUMBER: _ClassVar[int]
    SLOTS_FIELD_NUMBER: _ClassVar[int]
    LABELS_FIELD_NUMBER: _ClassVar[int]
    WEBHOOK_ID_FIELD_NUMBER: _ClassVar[int]
    RUNTIME_INFO_FIELD_NUMBER: _ClassVar[int]
    worker_name: str
    actions: _containers.RepeatedScalarFieldContainer[str]
    services: _containers.RepeatedScalarFieldContainer[str]
    slots: int
    labels: _containers.MessageMap[str, WorkerLabels]
    webhook_id: str
    runtime_info: RuntimeInfo
    def __init__(self, worker_name: _Optional[str] = ..., actions: _Optional[_Iterable[str]] = ..., services: _Optional[_Iterable[str]] = ..., slots: _Optional[int] = ..., labels: _Optional[_Mapping[str, WorkerLabels]] = ..., webhook_id: _Optional[str] = ..., runtime_info: _Optional[_Union[RuntimeInfo, _Mapping]] = ...) -> None: ...

class WorkerRegisterResponse(_message.Message):
    __slots__ = ("tenant_id", "worker_id", "worker_name")
    TENANT_ID_FIELD_NUMBER: _ClassVar[int]
    WORKER_ID_FIELD_NUMBER: _ClassVar[int]
    WORKER_NAME_FIELD_NUMBER: _ClassVar[int]
    tenant_id: str
    worker_id: str
    worker_name: str
    def __init__(self, tenant_id: _Optional[str] = ..., worker_id: _Optional[str] = ..., worker_name: _Optional[str] = ...) -> None: ...

class UpsertWorkerLabelsRequest(_message.Message):
    __slots__ = ("worker_id", "labels")
    class LabelsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: WorkerLabels
        def __init__(self, key: _Optional[str] = ..., value: _Optional[_Union[WorkerLabels, _Mapping]] = ...) -> None: ...
    WORKER_ID_FIELD_NUMBER: _ClassVar[int]
    LABELS_FIELD_NUMBER: _ClassVar[int]
    worker_id: str
    labels: _containers.MessageMap[str, WorkerLabels]
    def __init__(self, worker_id: _Optional[str] = ..., labels: _Optional[_Mapping[str, WorkerLabels]] = ...) -> None: ...

class UpsertWorkerLabelsResponse(_message.Message):
    __slots__ = ("tenant_id", "worker_id")
    TENANT_ID_FIELD_NUMBER: _ClassVar[int]
    WORKER_ID_FIELD_NUMBER: _ClassVar[int]
    tenant_id: str
    worker_id: str
    def __init__(self, tenant_id: _Optional[str] = ..., worker_id: _Optional[str] = ...) -> None: ...

class AssignedAction(_message.Message):
    __slots__ = ("tenant_id", "workflow_run_id", "get_group_key_run_id", "job_id", "job_name", "job_run_id", "task_id", "task_external_id", "action_id", "action_type", "action_payload", "step_name", "retry_count", "additional_metadata", "child_workflow_index", "child_workflow_key", "parent_workflow_run_id", "priority", "workflow_id", "workflow_version_id")
    TENANT_ID_FIELD_NUMBER: _ClassVar[int]
    WORKFLOW_RUN_ID_FIELD_NUMBER: _ClassVar[int]
    GET_GROUP_KEY_RUN_ID_FIELD_NUMBER: _ClassVar[int]
    JOB_ID_FIELD_NUMBER: _ClassVar[int]
    JOB_NAME_FIELD_NUMBER: _ClassVar[int]
    JOB_RUN_ID_FIELD_NUMBER: _ClassVar[int]
    TASK_ID_FIELD_NUMBER: _ClassVar[int]
    TASK_EXTERNAL_ID_FIELD_NUMBER: _ClassVar[int]
    ACTION_ID_FIELD_NUMBER: _ClassVar[int]
    ACTION_TYPE_FIELD_NUMBER: _ClassVar[int]
    ACTION_PAYLOAD_FIELD_NUMBER: _ClassVar[int]
    STEP_NAME_FIELD_NUMBER: _ClassVar[int]
    RETRY_COUNT_FIELD_NUMBER: _ClassVar[int]
    ADDITIONAL_METADATA_FIELD_NUMBER: _ClassVar[int]
    CHILD_WORKFLOW_INDEX_FIELD_NUMBER: _ClassVar[int]
    CHILD_WORKFLOW_KEY_FIELD_NUMBER: _ClassVar[int]
    PARENT_WORKFLOW_RUN_ID_FIELD_NUMBER: _ClassVar[int]
    PRIORITY_FIELD_NUMBER: _ClassVar[int]
    WORKFLOW_ID_FIELD_NUMBER: _ClassVar[int]
    WORKFLOW_VERSION_ID_FIELD_NUMBER: _ClassVar[int]
    tenant_id: str
    workflow_run_id: str
    get_group_key_run_id: str
    job_id: str
    job_name: str
    job_run_id: str
    task_id: str
    task_external_id: str
    action_id: str
    action_type: ActionType
    action_payload: str
    step_name: str
    retry_count: int
    additional_metadata: str
    child_workflow_index: int
    child_workflow_key: str
    parent_workflow_run_id: str
    priority: int
    workflow_id: str
    workflow_version_id: str
    def __init__(self, tenant_id: _Optional[str] = ..., workflow_run_id: _Optional[str] = ..., get_group_key_run_id: _Optional[str] = ..., job_id: _Optional[str] = ..., job_name: _Optional[str] = ..., job_run_id: _Optional[str] = ..., task_id: _Optional[str] = ..., task_external_id: _Optional[str] = ..., action_id: _Optional[str] = ..., action_type: _Optional[_Union[ActionType, str]] = ..., action_payload: _Optional[str] = ..., step_name: _Optional[str] = ..., retry_count: _Optional[int] = ..., additional_metadata: _Optional[str] = ..., child_workflow_index: _Optional[int] = ..., child_workflow_key: _Optional[str] = ..., parent_workflow_run_id: _Optional[str] = ..., priority: _Optional[int] = ..., workflow_id: _Optional[str] = ..., workflow_version_id: _Optional[str] = ...) -> None: ...

class WorkerListenRequest(_message.Message):
    __slots__ = ("worker_id",)
    WORKER_ID_FIELD_NUMBER: _ClassVar[int]
    worker_id: str
    def __init__(self, worker_id: _Optional[str] = ...) -> None: ...

class WorkerUnsubscribeRequest(_message.Message):
    __slots__ = ("worker_id",)
    WORKER_ID_FIELD_NUMBER: _ClassVar[int]
    worker_id: str
    def __init__(self, worker_id: _Optional[str] = ...) -> None: ...

class WorkerUnsubscribeResponse(_message.Message):
    __slots__ = ("tenant_id", "worker_id")
    TENANT_ID_FIELD_NUMBER: _ClassVar[int]
    WORKER_ID_FIELD_NUMBER: _ClassVar[int]
    tenant_id: str
    worker_id: str
    def __init__(self, tenant_id: _Optional[str] = ..., worker_id: _Optional[str] = ...) -> None: ...

class GroupKeyActionEvent(_message.Message):
    __slots__ = ("worker_id", "workflow_run_id", "get_group_key_run_id", "action_id", "event_timestamp", "event_type", "event_payload")
    WORKER_ID_FIELD_NUMBER: _ClassVar[int]
    WORKFLOW_RUN_ID_FIELD_NUMBER: _ClassVar[int]
    GET_GROUP_KEY_RUN_ID_FIELD_NUMBER: _ClassVar[int]
    ACTION_ID_FIELD_NUMBER: _ClassVar[int]
    EVENT_TIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    EVENT_TYPE_FIELD_NUMBER: _ClassVar[int]
    EVENT_PAYLOAD_FIELD_NUMBER: _ClassVar[int]
    worker_id: str
    workflow_run_id: str
    get_group_key_run_id: str
    action_id: str
    event_timestamp: _timestamp_pb2.Timestamp
    event_type: GroupKeyActionEventType
    event_payload: str
    def __init__(self, worker_id: _Optional[str] = ..., workflow_run_id: _Optional[str] = ..., get_group_key_run_id: _Optional[str] = ..., action_id: _Optional[str] = ..., event_timestamp: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., event_type: _Optional[_Union[GroupKeyActionEventType, str]] = ..., event_payload: _Optional[str] = ...) -> None: ...

class StepActionEvent(_message.Message):
    __slots__ = ("worker_id", "job_id", "job_run_id", "task_id", "task_external_id", "action_id", "event_timestamp", "event_type", "event_payload", "retry_count", "should_not_retry")
    WORKER_ID_FIELD_NUMBER: _ClassVar[int]
    JOB_ID_FIELD_NUMBER: _ClassVar[int]
    JOB_RUN_ID_FIELD_NUMBER: _ClassVar[int]
    TASK_ID_FIELD_NUMBER: _ClassVar[int]
    TASK_EXTERNAL_ID_FIELD_NUMBER: _ClassVar[int]
    ACTION_ID_FIELD_NUMBER: _ClassVar[int]
    EVENT_TIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    EVENT_TYPE_FIELD_NUMBER: _ClassVar[int]
    EVENT_PAYLOAD_FIELD_NUMBER: _ClassVar[int]
    RETRY_COUNT_FIELD_NUMBER: _ClassVar[int]
    SHOULD_NOT_RETRY_FIELD_NUMBER: _ClassVar[int]
    worker_id: str
    job_id: str
    job_run_id: str
    task_id: str
    task_external_id: str
    action_id: str
    event_timestamp: _timestamp_pb2.Timestamp
    event_type: StepActionEventType
    event_payload: str
    retry_count: int
    should_not_retry: bool
    def __init__(self, worker_id: _Optional[str] = ..., job_id: _Optional[str] = ..., job_run_id: _Optional[str] = ..., task_id: _Optional[str] = ..., task_external_id: _Optional[str] = ..., action_id: _Optional[str] = ..., event_timestamp: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., event_type: _Optional[_Union[StepActionEventType, str]] = ..., event_payload: _Optional[str] = ..., retry_count: _Optional[int] = ..., should_not_retry: bool = ...) -> None: ...

class ActionEventResponse(_message.Message):
    __slots__ = ("tenant_id", "worker_id")
    TENANT_ID_FIELD_NUMBER: _ClassVar[int]
    WORKER_ID_FIELD_NUMBER: _ClassVar[int]
    tenant_id: str
    worker_id: str
    def __init__(self, tenant_id: _Optional[str] = ..., worker_id: _Optional[str] = ...) -> None: ...

class SubscribeToWorkflowEventsRequest(_message.Message):
    __slots__ = ("workflow_run_id", "additional_meta_key", "additional_meta_value")
    WORKFLOW_RUN_ID_FIELD_NUMBER: _ClassVar[int]
    ADDITIONAL_META_KEY_FIELD_NUMBER: _ClassVar[int]
    ADDITIONAL_META_VALUE_FIELD_NUMBER: _ClassVar[int]
    workflow_run_id: str
    additional_meta_key: str
    additional_meta_value: str
    def __init__(self, workflow_run_id: _Optional[str] = ..., additional_meta_key: _Optional[str] = ..., additional_meta_value: _Optional[str] = ...) -> None: ...

class SubscribeToWorkflowRunsRequest(_message.Message):
    __slots__ = ("workflow_run_id",)
    WORKFLOW_RUN_ID_FIELD_NUMBER: _ClassVar[int]
    workflow_run_id: str
    def __init__(self, workflow_run_id: _Optional[str] = ...) -> None: ...

class WorkflowEvent(_message.Message):
    __slots__ = ("workflow_run_id", "resource_type", "event_type", "resource_id", "event_timestamp", "event_payload", "hangup", "step_retries", "retry_count", "event_index")
    WORKFLOW_RUN_ID_FIELD_NUMBER: _ClassVar[int]
    RESOURCE_TYPE_FIELD_NUMBER: _ClassVar[int]
    EVENT_TYPE_FIELD_NUMBER: _ClassVar[int]
    RESOURCE_ID_FIELD_NUMBER: _ClassVar[int]
    EVENT_TIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    EVENT_PAYLOAD_FIELD_NUMBER: _ClassVar[int]
    HANGUP_FIELD_NUMBER: _ClassVar[int]
    STEP_RETRIES_FIELD_NUMBER: _ClassVar[int]
    RETRY_COUNT_FIELD_NUMBER: _ClassVar[int]
    EVENT_INDEX_FIELD_NUMBER: _ClassVar[int]
    workflow_run_id: str
    resource_type: ResourceType
    event_type: ResourceEventType
    resource_id: str
    event_timestamp: _timestamp_pb2.Timestamp
    event_payload: str
    hangup: bool
    step_retries: int
    retry_count: int
    event_index: int
    def __init__(self, workflow_run_id: _Optional[str] = ..., resource_type: _Optional[_Union[ResourceType, str]] = ..., event_type: _Optional[_Union[ResourceEventType, str]] = ..., resource_id: _Optional[str] = ..., event_timestamp: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., event_payload: _Optional[str] = ..., hangup: bool = ..., step_retries: _Optional[int] = ..., retry_count: _Optional[int] = ..., event_index: _Optional[int] = ...) -> None: ...

class WorkflowRunEvent(_message.Message):
    __slots__ = ("workflow_run_id", "event_type", "event_timestamp", "results")
    WORKFLOW_RUN_ID_FIELD_NUMBER: _ClassVar[int]
    EVENT_TYPE_FIELD_NUMBER: _ClassVar[int]
    EVENT_TIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    RESULTS_FIELD_NUMBER: _ClassVar[int]
    workflow_run_id: str
    event_type: WorkflowRunEventType
    event_timestamp: _timestamp_pb2.Timestamp
    results: _containers.RepeatedCompositeFieldContainer[StepRunResult]
    def __init__(self, workflow_run_id: _Optional[str] = ..., event_type: _Optional[_Union[WorkflowRunEventType, str]] = ..., event_timestamp: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., results: _Optional[_Iterable[_Union[StepRunResult, _Mapping]]] = ...) -> None: ...

class StepRunResult(_message.Message):
    __slots__ = ("step_run_id", "step_readable_id", "job_run_id", "error", "output")
    STEP_RUN_ID_FIELD_NUMBER: _ClassVar[int]
    STEP_READABLE_ID_FIELD_NUMBER: _ClassVar[int]
    JOB_RUN_ID_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    OUTPUT_FIELD_NUMBER: _ClassVar[int]
    step_run_id: str
    step_readable_id: str
    job_run_id: str
    error: str
    output: str
    def __init__(self, step_run_id: _Optional[str] = ..., step_readable_id: _Optional[str] = ..., job_run_id: _Optional[str] = ..., error: _Optional[str] = ..., output: _Optional[str] = ...) -> None: ...

class OverridesData(_message.Message):
    __slots__ = ("step_run_id", "path", "value", "caller_filename")
    STEP_RUN_ID_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    VALUE_FIELD_NUMBER: _ClassVar[int]
    CALLER_FILENAME_FIELD_NUMBER: _ClassVar[int]
    step_run_id: str
    path: str
    value: str
    caller_filename: str
    def __init__(self, step_run_id: _Optional[str] = ..., path: _Optional[str] = ..., value: _Optional[str] = ..., caller_filename: _Optional[str] = ...) -> None: ...

class OverridesDataResponse(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class HeartbeatRequest(_message.Message):
    __slots__ = ("worker_id", "heartbeat_at")
    WORKER_ID_FIELD_NUMBER: _ClassVar[int]
    HEARTBEAT_AT_FIELD_NUMBER: _ClassVar[int]
    worker_id: str
    heartbeat_at: _timestamp_pb2.Timestamp
    def __init__(self, worker_id: _Optional[str] = ..., heartbeat_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class HeartbeatResponse(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class RefreshTimeoutRequest(_message.Message):
    __slots__ = ("step_run_id", "increment_timeout_by")
    STEP_RUN_ID_FIELD_NUMBER: _ClassVar[int]
    INCREMENT_TIMEOUT_BY_FIELD_NUMBER: _ClassVar[int]
    step_run_id: str
    increment_timeout_by: str
    def __init__(self, step_run_id: _Optional[str] = ..., increment_timeout_by: _Optional[str] = ...) -> None: ...

class RefreshTimeoutResponse(_message.Message):
    __slots__ = ("timeout_at",)
    TIMEOUT_AT_FIELD_NUMBER: _ClassVar[int]
    timeout_at: _timestamp_pb2.Timestamp
    def __init__(self, timeout_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class ReleaseSlotRequest(_message.Message):
    __slots__ = ("step_run_id",)
    STEP_RUN_ID_FIELD_NUMBER: _ClassVar[int]
    step_run_id: str
    def __init__(self, step_run_id: _Optional[str] = ...) -> None: ...

class ReleaseSlotResponse(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...
