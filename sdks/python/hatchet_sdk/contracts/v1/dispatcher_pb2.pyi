from hatchet_sdk.contracts.v1.shared import condition_pb2 as _condition_pb2
from hatchet_sdk.contracts.v1.shared import trigger_pb2 as _trigger_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class DurableTaskEventKind(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    DURABLE_TASK_TRIGGER_KIND_UNSPECIFIED: _ClassVar[DurableTaskEventKind]
    DURABLE_TASK_TRIGGER_KIND_RUN: _ClassVar[DurableTaskEventKind]
    DURABLE_TASK_TRIGGER_KIND_WAIT_FOR: _ClassVar[DurableTaskEventKind]
    DURABLE_TASK_TRIGGER_KIND_MEMO: _ClassVar[DurableTaskEventKind]
DURABLE_TASK_TRIGGER_KIND_UNSPECIFIED: DurableTaskEventKind
DURABLE_TASK_TRIGGER_KIND_RUN: DurableTaskEventKind
DURABLE_TASK_TRIGGER_KIND_WAIT_FOR: DurableTaskEventKind
DURABLE_TASK_TRIGGER_KIND_MEMO: DurableTaskEventKind

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

class DurableTaskEventRequest(_message.Message):
    __slots__ = ("invocation_count", "durable_task_external_id", "kind", "payload", "wait_for_conditions", "trigger_opts")
    INVOCATION_COUNT_FIELD_NUMBER: _ClassVar[int]
    DURABLE_TASK_EXTERNAL_ID_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    PAYLOAD_FIELD_NUMBER: _ClassVar[int]
    WAIT_FOR_CONDITIONS_FIELD_NUMBER: _ClassVar[int]
    TRIGGER_OPTS_FIELD_NUMBER: _ClassVar[int]
    invocation_count: int
    durable_task_external_id: str
    kind: DurableTaskEventKind
    payload: bytes
    wait_for_conditions: _condition_pb2.DurableEventListenerConditions
    trigger_opts: _trigger_pb2.TriggerWorkflowRequest
    def __init__(self, invocation_count: _Optional[int] = ..., durable_task_external_id: _Optional[str] = ..., kind: _Optional[_Union[DurableTaskEventKind, str]] = ..., payload: _Optional[bytes] = ..., wait_for_conditions: _Optional[_Union[_condition_pb2.DurableEventListenerConditions, _Mapping]] = ..., trigger_opts: _Optional[_Union[_trigger_pb2.TriggerWorkflowRequest, _Mapping]] = ...) -> None: ...

class DurableTaskEventAckResponse(_message.Message):
    __slots__ = ("invocation_count", "durable_task_external_id", "node_id")
    INVOCATION_COUNT_FIELD_NUMBER: _ClassVar[int]
    DURABLE_TASK_EXTERNAL_ID_FIELD_NUMBER: _ClassVar[int]
    NODE_ID_FIELD_NUMBER: _ClassVar[int]
    invocation_count: int
    durable_task_external_id: str
    node_id: int
    def __init__(self, invocation_count: _Optional[int] = ..., durable_task_external_id: _Optional[str] = ..., node_id: _Optional[int] = ...) -> None: ...

class DurableTaskCallbackCompletedResponse(_message.Message):
    __slots__ = ("durable_task_external_id", "node_id", "payload")
    DURABLE_TASK_EXTERNAL_ID_FIELD_NUMBER: _ClassVar[int]
    NODE_ID_FIELD_NUMBER: _ClassVar[int]
    PAYLOAD_FIELD_NUMBER: _ClassVar[int]
    durable_task_external_id: str
    node_id: int
    payload: bytes
    def __init__(self, durable_task_external_id: _Optional[str] = ..., node_id: _Optional[int] = ..., payload: _Optional[bytes] = ...) -> None: ...

class DurableTaskEvictInvocationRequest(_message.Message):
    __slots__ = ("invocation_count", "durable_task_external_id")
    INVOCATION_COUNT_FIELD_NUMBER: _ClassVar[int]
    DURABLE_TASK_EXTERNAL_ID_FIELD_NUMBER: _ClassVar[int]
    invocation_count: int
    durable_task_external_id: str
    def __init__(self, invocation_count: _Optional[int] = ..., durable_task_external_id: _Optional[str] = ...) -> None: ...

class DurableTaskAwaitedCallback(_message.Message):
    __slots__ = ("durable_task_external_id", "node_id")
    DURABLE_TASK_EXTERNAL_ID_FIELD_NUMBER: _ClassVar[int]
    NODE_ID_FIELD_NUMBER: _ClassVar[int]
    durable_task_external_id: str
    node_id: int
    def __init__(self, durable_task_external_id: _Optional[str] = ..., node_id: _Optional[int] = ...) -> None: ...

class DurableTaskWorkerStatusRequest(_message.Message):
    __slots__ = ("worker_id", "node_id", "branch_id", "waiting_callbacks")
    WORKER_ID_FIELD_NUMBER: _ClassVar[int]
    NODE_ID_FIELD_NUMBER: _ClassVar[int]
    BRANCH_ID_FIELD_NUMBER: _ClassVar[int]
    WAITING_CALLBACKS_FIELD_NUMBER: _ClassVar[int]
    worker_id: str
    node_id: int
    branch_id: int
    waiting_callbacks: _containers.RepeatedCompositeFieldContainer[DurableTaskAwaitedCallback]
    def __init__(self, worker_id: _Optional[str] = ..., node_id: _Optional[int] = ..., branch_id: _Optional[int] = ..., waiting_callbacks: _Optional[_Iterable[_Union[DurableTaskAwaitedCallback, _Mapping]]] = ...) -> None: ...

class DurableTaskRequest(_message.Message):
    __slots__ = ("register_worker", "event", "evict_invocation", "worker_status")
    REGISTER_WORKER_FIELD_NUMBER: _ClassVar[int]
    EVENT_FIELD_NUMBER: _ClassVar[int]
    EVICT_INVOCATION_FIELD_NUMBER: _ClassVar[int]
    WORKER_STATUS_FIELD_NUMBER: _ClassVar[int]
    register_worker: DurableTaskRequestRegisterWorker
    event: DurableTaskEventRequest
    evict_invocation: DurableTaskEvictInvocationRequest
    worker_status: DurableTaskWorkerStatusRequest
    def __init__(self, register_worker: _Optional[_Union[DurableTaskRequestRegisterWorker, _Mapping]] = ..., event: _Optional[_Union[DurableTaskEventRequest, _Mapping]] = ..., evict_invocation: _Optional[_Union[DurableTaskEvictInvocationRequest, _Mapping]] = ..., worker_status: _Optional[_Union[DurableTaskWorkerStatusRequest, _Mapping]] = ...) -> None: ...

class DurableTaskErrorResponse(_message.Message):
    __slots__ = ("durable_task_external_id", "invocation_count", "node_id", "error_message")
    DURABLE_TASK_EXTERNAL_ID_FIELD_NUMBER: _ClassVar[int]
    INVOCATION_COUNT_FIELD_NUMBER: _ClassVar[int]
    NODE_ID_FIELD_NUMBER: _ClassVar[int]
    ERROR_MESSAGE_FIELD_NUMBER: _ClassVar[int]
    durable_task_external_id: str
    invocation_count: int
    node_id: int
    error_message: str
    def __init__(self, durable_task_external_id: _Optional[str] = ..., invocation_count: _Optional[int] = ..., node_id: _Optional[int] = ..., error_message: _Optional[str] = ...) -> None: ...

class DurableTaskResponse(_message.Message):
    __slots__ = ("register_worker", "trigger_ack", "callback_completed", "error")
    REGISTER_WORKER_FIELD_NUMBER: _ClassVar[int]
    TRIGGER_ACK_FIELD_NUMBER: _ClassVar[int]
    CALLBACK_COMPLETED_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    register_worker: DurableTaskResponseRegisterWorker
    trigger_ack: DurableTaskEventAckResponse
    callback_completed: DurableTaskCallbackCompletedResponse
    error: DurableTaskErrorResponse
    def __init__(self, register_worker: _Optional[_Union[DurableTaskResponseRegisterWorker, _Mapping]] = ..., trigger_ack: _Optional[_Union[DurableTaskEventAckResponse, _Mapping]] = ..., callback_completed: _Optional[_Union[DurableTaskCallbackCompletedResponse, _Mapping]] = ..., error: _Optional[_Union[DurableTaskErrorResponse, _Mapping]] = ...) -> None: ...

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
