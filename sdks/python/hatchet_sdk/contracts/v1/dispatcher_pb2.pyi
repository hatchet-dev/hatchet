from hatchet_sdk.contracts.v1.shared import condition_pb2 as _condition_pb2
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from typing import ClassVar as _ClassVar, Mapping as _Mapping, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class RegisterDurableEventRequest(_message.Message):
    __slots__ = ("task_id", "workflow_run_id", "conditions")
    TASK_ID_FIELD_NUMBER: _ClassVar[int]
    WORKFLOW_RUN_ID_FIELD_NUMBER: _ClassVar[int]
    CONDITIONS_FIELD_NUMBER: _ClassVar[int]
    task_id: str
    workflow_run_id: str
    conditions: _condition_pb2.DurableEventListenerConditions
    def __init__(self, task_id: _Optional[str] = ..., workflow_run_id: _Optional[str] = ..., conditions: _Optional[_Union[_condition_pb2.DurableEventListenerConditions, _Mapping]] = ...) -> None: ...

class RegisterDurableEventResponse(_message.Message):
    __slots__ = ("match_id",)
    MATCH_ID_FIELD_NUMBER: _ClassVar[int]
    match_id: int
    def __init__(self, match_id: _Optional[int] = ...) -> None: ...

class ListenForDurableEventRequest(_message.Message):
    __slots__ = ("task_id", "match_id")
    TASK_ID_FIELD_NUMBER: _ClassVar[int]
    MATCH_ID_FIELD_NUMBER: _ClassVar[int]
    task_id: str
    match_id: int
    def __init__(self, task_id: _Optional[str] = ..., match_id: _Optional[int] = ...) -> None: ...

class DurableEvent(_message.Message):
    __slots__ = ("task_id", "match_id", "data")
    TASK_ID_FIELD_NUMBER: _ClassVar[int]
    MATCH_ID_FIELD_NUMBER: _ClassVar[int]
    DATA_FIELD_NUMBER: _ClassVar[int]
    task_id: str
    match_id: int
    data: bytes
    def __init__(self, task_id: _Optional[str] = ..., match_id: _Optional[int] = ..., data: _Optional[bytes] = ...) -> None: ...
