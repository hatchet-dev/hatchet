from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from typing import ClassVar as _ClassVar, Iterable as _Iterable, Mapping as _Mapping, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Event(_message.Message):
    __slots__ = ("tenantId", "eventId", "key", "payload", "eventTimestamp")
    TENANTID_FIELD_NUMBER: _ClassVar[int]
    EVENTID_FIELD_NUMBER: _ClassVar[int]
    KEY_FIELD_NUMBER: _ClassVar[int]
    PAYLOAD_FIELD_NUMBER: _ClassVar[int]
    EVENTTIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    tenantId: str
    eventId: str
    key: str
    payload: str
    eventTimestamp: _timestamp_pb2.Timestamp
    def __init__(self, tenantId: _Optional[str] = ..., eventId: _Optional[str] = ..., key: _Optional[str] = ..., payload: _Optional[str] = ..., eventTimestamp: _Optional[_Union[_timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class PushEventRequest(_message.Message):
    __slots__ = ("tenantId", "key", "payload", "eventTimestamp")
    TENANTID_FIELD_NUMBER: _ClassVar[int]
    KEY_FIELD_NUMBER: _ClassVar[int]
    PAYLOAD_FIELD_NUMBER: _ClassVar[int]
    EVENTTIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    tenantId: str
    key: str
    payload: str
    eventTimestamp: _timestamp_pb2.Timestamp
    def __init__(self, tenantId: _Optional[str] = ..., key: _Optional[str] = ..., payload: _Optional[str] = ..., eventTimestamp: _Optional[_Union[_timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class ListEventRequest(_message.Message):
    __slots__ = ("tenantId", "offset", "key")
    TENANTID_FIELD_NUMBER: _ClassVar[int]
    OFFSET_FIELD_NUMBER: _ClassVar[int]
    KEY_FIELD_NUMBER: _ClassVar[int]
    tenantId: str
    offset: int
    key: str
    def __init__(self, tenantId: _Optional[str] = ..., offset: _Optional[int] = ..., key: _Optional[str] = ...) -> None: ...

class ListEventResponse(_message.Message):
    __slots__ = ("events",)
    EVENTS_FIELD_NUMBER: _ClassVar[int]
    events: _containers.RepeatedCompositeFieldContainer[Event]
    def __init__(self, events: _Optional[_Iterable[_Union[Event, _Mapping]]] = ...) -> None: ...

class ReplayEventRequest(_message.Message):
    __slots__ = ("tenantId", "eventId")
    TENANTID_FIELD_NUMBER: _ClassVar[int]
    EVENTID_FIELD_NUMBER: _ClassVar[int]
    tenantId: str
    eventId: str
    def __init__(self, tenantId: _Optional[str] = ..., eventId: _Optional[str] = ...) -> None: ...
