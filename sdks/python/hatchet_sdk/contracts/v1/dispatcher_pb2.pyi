from hatchet_sdk.contracts.v1.shared import condition_pb2 as _condition_pb2
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from typing import ClassVar as _ClassVar, Mapping as _Mapping, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

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
