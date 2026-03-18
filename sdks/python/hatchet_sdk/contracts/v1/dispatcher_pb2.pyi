from hatchet_sdk.contracts.v1.shared import condition_pb2 as _condition_pb2
from hatchet_sdk.contracts.v1.shared import trigger_pb2 as _trigger_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class DurableTaskErrorType(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    DURABLE_TASK_ERROR_TYPE_UNSPECIFIED: _ClassVar[DurableTaskErrorType]
    DURABLE_TASK_ERROR_TYPE_NONDETERMINISM: _ClassVar[DurableTaskErrorType]
DURABLE_TASK_ERROR_TYPE_UNSPECIFIED: DurableTaskErrorType
DURABLE_TASK_ERROR_TYPE_NONDETERMINISM: DurableTaskErrorType

class DurableTaskRequestRegisterWorker(_message.Message):
    __slots__ = ("worker_id",)
    WORKER_ID_FIELD_NUMBER: _ClassVar[int]
    worker_id: str
    def __init__(self, worker_id: _Optional[str] = ...) -> None: ...

class DurableTaskResponseRegisterWorker(_message.Message):
    __slots__ = ("worker_id",)
    WORKER_ID_FIELD_NUMBER: _ClassVar[int]
    worker_id: str
    def __init__(self, worker_id: _Optional[str] = ...) -> None: ...

class DurableEventLogEntryRef(_message.Message):
    __slots__ = ("durable_task_external_id", "invocation_count", "branch_id", "node_id")
    DURABLE_TASK_EXTERNAL_ID_FIELD_NUMBER: _ClassVar[int]
    INVOCATION_COUNT_FIELD_NUMBER: _ClassVar[int]
    BRANCH_ID_FIELD_NUMBER: _ClassVar[int]
    NODE_ID_FIELD_NUMBER: _ClassVar[int]
    durable_task_external_id: str
    invocation_count: int
    branch_id: int
    node_id: int
    def __init__(self, durable_task_external_id: _Optional[str] = ..., invocation_count: _Optional[int] = ..., branch_id: _Optional[int] = ..., node_id: _Optional[int] = ...) -> None: ...

class DurableTaskRunAckEntry(_message.Message):
    __slots__ = ("node_id", "branch_id", "workflow_run_external_id")
    NODE_ID_FIELD_NUMBER: _ClassVar[int]
    BRANCH_ID_FIELD_NUMBER: _ClassVar[int]
    WORKFLOW_RUN_EXTERNAL_ID_FIELD_NUMBER: _ClassVar[int]
    node_id: int
    branch_id: int
    workflow_run_external_id: str
    def __init__(self, node_id: _Optional[int] = ..., branch_id: _Optional[int] = ..., workflow_run_external_id: _Optional[str] = ...) -> None: ...

class DurableTaskEventMemoAckResponse(_message.Message):
    __slots__ = ("ref", "memo_already_existed", "memo_result_payload")
    REF_FIELD_NUMBER: _ClassVar[int]
    MEMO_ALREADY_EXISTED_FIELD_NUMBER: _ClassVar[int]
    MEMO_RESULT_PAYLOAD_FIELD_NUMBER: _ClassVar[int]
    ref: DurableEventLogEntryRef
    memo_already_existed: bool
    memo_result_payload: bytes
    def __init__(self, ref: _Optional[_Union[DurableEventLogEntryRef, _Mapping]] = ..., memo_already_existed: bool = ..., memo_result_payload: _Optional[bytes] = ...) -> None: ...

class DurableTaskEventTriggerRunsAckResponse(_message.Message):
    __slots__ = ("durable_task_external_id", "invocation_count", "run_entries")
    DURABLE_TASK_EXTERNAL_ID_FIELD_NUMBER: _ClassVar[int]
    INVOCATION_COUNT_FIELD_NUMBER: _ClassVar[int]
    RUN_ENTRIES_FIELD_NUMBER: _ClassVar[int]
    durable_task_external_id: str
    invocation_count: int
    run_entries: _containers.RepeatedCompositeFieldContainer[DurableTaskRunAckEntry]
    def __init__(self, durable_task_external_id: _Optional[str] = ..., invocation_count: _Optional[int] = ..., run_entries: _Optional[_Iterable[_Union[DurableTaskRunAckEntry, _Mapping]]] = ...) -> None: ...

class DurableTaskEventWaitForAckResponse(_message.Message):
    __slots__ = ("ref",)
    REF_FIELD_NUMBER: _ClassVar[int]
    ref: DurableEventLogEntryRef
    def __init__(self, ref: _Optional[_Union[DurableEventLogEntryRef, _Mapping]] = ...) -> None: ...

class DurableTaskEventLogEntryCompletedResponse(_message.Message):
    __slots__ = ("ref", "payload")
    REF_FIELD_NUMBER: _ClassVar[int]
    PAYLOAD_FIELD_NUMBER: _ClassVar[int]
    ref: DurableEventLogEntryRef
    payload: bytes
    def __init__(self, ref: _Optional[_Union[DurableEventLogEntryRef, _Mapping]] = ..., payload: _Optional[bytes] = ...) -> None: ...

class DurableTaskEvictInvocationRequest(_message.Message):
    __slots__ = ("invocation_count", "durable_task_external_id", "reason")
    INVOCATION_COUNT_FIELD_NUMBER: _ClassVar[int]
    DURABLE_TASK_EXTERNAL_ID_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    invocation_count: int
    durable_task_external_id: str
    reason: str
    def __init__(self, invocation_count: _Optional[int] = ..., durable_task_external_id: _Optional[str] = ..., reason: _Optional[str] = ...) -> None: ...

class DurableTaskEvictionAckResponse(_message.Message):
    __slots__ = ("invocation_count", "durable_task_external_id")
    INVOCATION_COUNT_FIELD_NUMBER: _ClassVar[int]
    DURABLE_TASK_EXTERNAL_ID_FIELD_NUMBER: _ClassVar[int]
    invocation_count: int
    durable_task_external_id: str
    def __init__(self, invocation_count: _Optional[int] = ..., durable_task_external_id: _Optional[str] = ...) -> None: ...

class DurableTaskAwaitedCompletedEntry(_message.Message):
    __slots__ = ("durable_task_external_id", "branch_id", "node_id", "invocation_count")
    DURABLE_TASK_EXTERNAL_ID_FIELD_NUMBER: _ClassVar[int]
    BRANCH_ID_FIELD_NUMBER: _ClassVar[int]
    NODE_ID_FIELD_NUMBER: _ClassVar[int]
    INVOCATION_COUNT_FIELD_NUMBER: _ClassVar[int]
    durable_task_external_id: str
    branch_id: int
    node_id: int
    invocation_count: int
    def __init__(self, durable_task_external_id: _Optional[str] = ..., branch_id: _Optional[int] = ..., node_id: _Optional[int] = ..., invocation_count: _Optional[int] = ...) -> None: ...

class DurableTaskServerEvictNotice(_message.Message):
    __slots__ = ("durable_task_external_id", "invocation_count", "reason")
    DURABLE_TASK_EXTERNAL_ID_FIELD_NUMBER: _ClassVar[int]
    INVOCATION_COUNT_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    durable_task_external_id: str
    invocation_count: int
    reason: str
    def __init__(self, durable_task_external_id: _Optional[str] = ..., invocation_count: _Optional[int] = ..., reason: _Optional[str] = ...) -> None: ...

class DurableTaskWorkerStatusRequest(_message.Message):
    __slots__ = ("worker_id", "waiting_entries")
    WORKER_ID_FIELD_NUMBER: _ClassVar[int]
    WAITING_ENTRIES_FIELD_NUMBER: _ClassVar[int]
    worker_id: str
    waiting_entries: _containers.RepeatedCompositeFieldContainer[DurableTaskAwaitedCompletedEntry]
    def __init__(self, worker_id: _Optional[str] = ..., waiting_entries: _Optional[_Iterable[_Union[DurableTaskAwaitedCompletedEntry, _Mapping]]] = ...) -> None: ...

class DurableTaskCompleteMemoRequest(_message.Message):
    __slots__ = ("ref", "payload", "memo_key")
    REF_FIELD_NUMBER: _ClassVar[int]
    PAYLOAD_FIELD_NUMBER: _ClassVar[int]
    MEMO_KEY_FIELD_NUMBER: _ClassVar[int]
    ref: DurableEventLogEntryRef
    payload: bytes
    memo_key: bytes
    def __init__(self, ref: _Optional[_Union[DurableEventLogEntryRef, _Mapping]] = ..., payload: _Optional[bytes] = ..., memo_key: _Optional[bytes] = ...) -> None: ...

class DurableTaskMemoRequest(_message.Message):
    __slots__ = ("invocation_count", "durable_task_external_id", "key", "payload")
    INVOCATION_COUNT_FIELD_NUMBER: _ClassVar[int]
    DURABLE_TASK_EXTERNAL_ID_FIELD_NUMBER: _ClassVar[int]
    KEY_FIELD_NUMBER: _ClassVar[int]
    PAYLOAD_FIELD_NUMBER: _ClassVar[int]
    invocation_count: int
    durable_task_external_id: str
    key: bytes
    payload: bytes
    def __init__(self, invocation_count: _Optional[int] = ..., durable_task_external_id: _Optional[str] = ..., key: _Optional[bytes] = ..., payload: _Optional[bytes] = ...) -> None: ...

class DurableTaskTriggerRunsRequest(_message.Message):
    __slots__ = ("invocation_count", "durable_task_external_id", "trigger_opts")
    INVOCATION_COUNT_FIELD_NUMBER: _ClassVar[int]
    DURABLE_TASK_EXTERNAL_ID_FIELD_NUMBER: _ClassVar[int]
    TRIGGER_OPTS_FIELD_NUMBER: _ClassVar[int]
    invocation_count: int
    durable_task_external_id: str
    trigger_opts: _containers.RepeatedCompositeFieldContainer[_trigger_pb2.TriggerWorkflowRequest]
    def __init__(self, invocation_count: _Optional[int] = ..., durable_task_external_id: _Optional[str] = ..., trigger_opts: _Optional[_Iterable[_Union[_trigger_pb2.TriggerWorkflowRequest, _Mapping]]] = ...) -> None: ...

class DurableTaskWaitForRequest(_message.Message):
    __slots__ = ("invocation_count", "durable_task_external_id", "wait_for_conditions")
    INVOCATION_COUNT_FIELD_NUMBER: _ClassVar[int]
    DURABLE_TASK_EXTERNAL_ID_FIELD_NUMBER: _ClassVar[int]
    WAIT_FOR_CONDITIONS_FIELD_NUMBER: _ClassVar[int]
    invocation_count: int
    durable_task_external_id: str
    wait_for_conditions: _condition_pb2.DurableEventListenerConditions
    def __init__(self, invocation_count: _Optional[int] = ..., durable_task_external_id: _Optional[str] = ..., wait_for_conditions: _Optional[_Union[_condition_pb2.DurableEventListenerConditions, _Mapping]] = ...) -> None: ...

class DurableTaskRequest(_message.Message):
    __slots__ = ("register_worker", "memo", "trigger_runs", "wait_for", "evict_invocation", "worker_status", "complete_memo")
    REGISTER_WORKER_FIELD_NUMBER: _ClassVar[int]
    MEMO_FIELD_NUMBER: _ClassVar[int]
    TRIGGER_RUNS_FIELD_NUMBER: _ClassVar[int]
    WAIT_FOR_FIELD_NUMBER: _ClassVar[int]
    EVICT_INVOCATION_FIELD_NUMBER: _ClassVar[int]
    WORKER_STATUS_FIELD_NUMBER: _ClassVar[int]
    COMPLETE_MEMO_FIELD_NUMBER: _ClassVar[int]
    register_worker: DurableTaskRequestRegisterWorker
    memo: DurableTaskMemoRequest
    trigger_runs: DurableTaskTriggerRunsRequest
    wait_for: DurableTaskWaitForRequest
    evict_invocation: DurableTaskEvictInvocationRequest
    worker_status: DurableTaskWorkerStatusRequest
    complete_memo: DurableTaskCompleteMemoRequest
    def __init__(self, register_worker: _Optional[_Union[DurableTaskRequestRegisterWorker, _Mapping]] = ..., memo: _Optional[_Union[DurableTaskMemoRequest, _Mapping]] = ..., trigger_runs: _Optional[_Union[DurableTaskTriggerRunsRequest, _Mapping]] = ..., wait_for: _Optional[_Union[DurableTaskWaitForRequest, _Mapping]] = ..., evict_invocation: _Optional[_Union[DurableTaskEvictInvocationRequest, _Mapping]] = ..., worker_status: _Optional[_Union[DurableTaskWorkerStatusRequest, _Mapping]] = ..., complete_memo: _Optional[_Union[DurableTaskCompleteMemoRequest, _Mapping]] = ...) -> None: ...

class DurableTaskErrorResponse(_message.Message):
    __slots__ = ("ref", "error_type", "error_message")
    REF_FIELD_NUMBER: _ClassVar[int]
    ERROR_TYPE_FIELD_NUMBER: _ClassVar[int]
    ERROR_MESSAGE_FIELD_NUMBER: _ClassVar[int]
    ref: DurableEventLogEntryRef
    error_type: DurableTaskErrorType
    error_message: str
    def __init__(self, ref: _Optional[_Union[DurableEventLogEntryRef, _Mapping]] = ..., error_type: _Optional[_Union[DurableTaskErrorType, str]] = ..., error_message: _Optional[str] = ...) -> None: ...

class DurableTaskResponse(_message.Message):
    __slots__ = ("register_worker", "memo_ack", "trigger_runs_ack", "wait_for_ack", "entry_completed", "error", "eviction_ack", "server_evict")
    REGISTER_WORKER_FIELD_NUMBER: _ClassVar[int]
    MEMO_ACK_FIELD_NUMBER: _ClassVar[int]
    TRIGGER_RUNS_ACK_FIELD_NUMBER: _ClassVar[int]
    WAIT_FOR_ACK_FIELD_NUMBER: _ClassVar[int]
    ENTRY_COMPLETED_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    EVICTION_ACK_FIELD_NUMBER: _ClassVar[int]
    SERVER_EVICT_FIELD_NUMBER: _ClassVar[int]
    register_worker: DurableTaskResponseRegisterWorker
    memo_ack: DurableTaskEventMemoAckResponse
    trigger_runs_ack: DurableTaskEventTriggerRunsAckResponse
    wait_for_ack: DurableTaskEventWaitForAckResponse
    entry_completed: DurableTaskEventLogEntryCompletedResponse
    error: DurableTaskErrorResponse
    eviction_ack: DurableTaskEvictionAckResponse
    server_evict: DurableTaskServerEvictNotice
    def __init__(self, register_worker: _Optional[_Union[DurableTaskResponseRegisterWorker, _Mapping]] = ..., memo_ack: _Optional[_Union[DurableTaskEventMemoAckResponse, _Mapping]] = ..., trigger_runs_ack: _Optional[_Union[DurableTaskEventTriggerRunsAckResponse, _Mapping]] = ..., wait_for_ack: _Optional[_Union[DurableTaskEventWaitForAckResponse, _Mapping]] = ..., entry_completed: _Optional[_Union[DurableTaskEventLogEntryCompletedResponse, _Mapping]] = ..., error: _Optional[_Union[DurableTaskErrorResponse, _Mapping]] = ..., eviction_ack: _Optional[_Union[DurableTaskEvictionAckResponse, _Mapping]] = ..., server_evict: _Optional[_Union[DurableTaskServerEvictNotice, _Mapping]] = ...) -> None: ...

class RegisterDurableEventRequest(_message.Message):
    __slots__ = ("task_id", "signal_key", "conditions")
    TASK_ID_FIELD_NUMBER: _ClassVar[int]
    SIGNAL_KEY_FIELD_NUMBER: _ClassVar[int]
    CONDITIONS_FIELD_NUMBER: _ClassVar[int]
    task_id: str
    signal_key: str
    conditions: _condition_pb2.DurableEventListenerConditions
    def __init__(self, task_id: _Optional[str] = ..., signal_key: _Optional[str] = ..., conditions: _Optional[_Union[_condition_pb2.DurableEventListenerConditions, _Mapping]] = ...) -> None: ...

class RegisterDurableEventResponse(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ListenForDurableEventRequest(_message.Message):
    __slots__ = ("task_id", "signal_key")
    TASK_ID_FIELD_NUMBER: _ClassVar[int]
    SIGNAL_KEY_FIELD_NUMBER: _ClassVar[int]
    task_id: str
    signal_key: str
    def __init__(self, task_id: _Optional[str] = ..., signal_key: _Optional[str] = ...) -> None: ...

class DurableEvent(_message.Message):
    __slots__ = ("task_id", "signal_key", "data")
    TASK_ID_FIELD_NUMBER: _ClassVar[int]
    SIGNAL_KEY_FIELD_NUMBER: _ClassVar[int]
    DATA_FIELD_NUMBER: _ClassVar[int]
    task_id: str
    signal_key: str
    data: bytes
    def __init__(self, task_id: _Optional[str] = ..., signal_key: _Optional[str] = ..., data: _Optional[bytes] = ...) -> None: ...
