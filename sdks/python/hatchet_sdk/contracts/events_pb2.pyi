from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from typing import ClassVar as _ClassVar, Iterable as _Iterable, Mapping as _Mapping, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Event(_message.Message):
    __slots__ = ("tenantId", "eventId", "key", "payload", "eventTimestamp", "additionalMetadata")
    TENANTID_FIELD_NUMBER: _ClassVar[int]
    EVENTID_FIELD_NUMBER: _ClassVar[int]
    KEY_FIELD_NUMBER: _ClassVar[int]
    PAYLOAD_FIELD_NUMBER: _ClassVar[int]
    EVENTTIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    ADDITIONALMETADATA_FIELD_NUMBER: _ClassVar[int]
    tenantId: str
    eventId: str
    key: str
    payload: str
    eventTimestamp: _timestamp_pb2.Timestamp
    additionalMetadata: str
    def __init__(self, tenantId: _Optional[str] = ..., eventId: _Optional[str] = ..., key: _Optional[str] = ..., payload: _Optional[str] = ..., eventTimestamp: _Optional[_Union[_timestamp_pb2.Timestamp, _Mapping]] = ..., additionalMetadata: _Optional[str] = ...) -> None: ...

class Events(_message.Message):
    __slots__ = ("events",)
    EVENTS_FIELD_NUMBER: _ClassVar[int]
    events: _containers.RepeatedCompositeFieldContainer[Event]
    def __init__(self, events: _Optional[_Iterable[_Union[Event, _Mapping]]] = ...) -> None: ...

class PutLogRequest(_message.Message):
    __slots__ = ("stepRunId", "createdAt", "message", "level", "metadata")
    STEPRUNID_FIELD_NUMBER: _ClassVar[int]
    CREATEDAT_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    LEVEL_FIELD_NUMBER: _ClassVar[int]
    METADATA_FIELD_NUMBER: _ClassVar[int]
    stepRunId: str
    createdAt: _timestamp_pb2.Timestamp
    message: str
    level: str
    metadata: str
    def __init__(self, stepRunId: _Optional[str] = ..., createdAt: _Optional[_Union[_timestamp_pb2.Timestamp, _Mapping]] = ..., message: _Optional[str] = ..., level: _Optional[str] = ..., metadata: _Optional[str] = ...) -> None: ...

class PutLogResponse(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class PutStreamEventRequest(_message.Message):
    __slots__ = ("stepRunId", "createdAt", "message", "metadata")
    STEPRUNID_FIELD_NUMBER: _ClassVar[int]
    CREATEDAT_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    METADATA_FIELD_NUMBER: _ClassVar[int]
    stepRunId: str
    createdAt: _timestamp_pb2.Timestamp
    message: bytes
    metadata: str
    def __init__(self, stepRunId: _Optional[str] = ..., createdAt: _Optional[_Union[_timestamp_pb2.Timestamp, _Mapping]] = ..., message: _Optional[bytes] = ..., metadata: _Optional[str] = ...) -> None: ...

class PutStreamEventResponse(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class BulkPushEventRequest(_message.Message):
    __slots__ = ("events",)
    EVENTS_FIELD_NUMBER: _ClassVar[int]
    events: _containers.RepeatedCompositeFieldContainer[PushEventRequest]
    def __init__(self, events: _Optional[_Iterable[_Union[PushEventRequest, _Mapping]]] = ...) -> None: ...

class PushEventRequest(_message.Message):
    __slots__ = ("key", "payload", "eventTimestamp", "additionalMetadata")
    KEY_FIELD_NUMBER: _ClassVar[int]
    PAYLOAD_FIELD_NUMBER: _ClassVar[int]
    EVENTTIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    ADDITIONALMETADATA_FIELD_NUMBER: _ClassVar[int]
    key: str
    payload: str
    eventTimestamp: _timestamp_pb2.Timestamp
    additionalMetadata: str
    def __init__(self, key: _Optional[str] = ..., payload: _Optional[str] = ..., eventTimestamp: _Optional[_Union[_timestamp_pb2.Timestamp, _Mapping]] = ..., additionalMetadata: _Optional[str] = ...) -> None: ...

class ReplayEventRequest(_message.Message):
    __slots__ = ("eventId",)
    EVENTID_FIELD_NUMBER: _ClassVar[int]
    eventId: str
    def __init__(self, eventId: _Optional[str] = ...) -> None: ...
