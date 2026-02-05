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

DESCRIPTOR: _descriptor.FileDescriptor

class Event(_message.Message):
    __slots__ = ("tenant_id", "event_id", "key", "payload", "event_timestamp", "additional_metadata", "scope")
    TENANT_ID_FIELD_NUMBER: _ClassVar[int]
    EVENT_ID_FIELD_NUMBER: _ClassVar[int]
    KEY_FIELD_NUMBER: _ClassVar[int]
    PAYLOAD_FIELD_NUMBER: _ClassVar[int]
    EVENT_TIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    ADDITIONAL_METADATA_FIELD_NUMBER: _ClassVar[int]
    SCOPE_FIELD_NUMBER: _ClassVar[int]
    tenant_id: str
    event_id: str
    key: str
    payload: str
    event_timestamp: _timestamp_pb2.Timestamp
    additional_metadata: str
    scope: str
    def __init__(self, tenant_id: _Optional[str] = ..., event_id: _Optional[str] = ..., key: _Optional[str] = ..., payload: _Optional[str] = ..., event_timestamp: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., additional_metadata: _Optional[str] = ..., scope: _Optional[str] = ...) -> None: ...

class Events(_message.Message):
    __slots__ = ("events",)
    EVENTS_FIELD_NUMBER: _ClassVar[int]
    events: _containers.RepeatedCompositeFieldContainer[Event]
    def __init__(self, events: _Optional[_Iterable[_Union[Event, _Mapping]]] = ...) -> None: ...

class PutLogRequest(_message.Message):
    __slots__ = ("task_run_external_id", "created_at", "message", "level", "metadata", "task_retry_count")
    TASK_RUN_EXTERNAL_ID_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    LEVEL_FIELD_NUMBER: _ClassVar[int]
    METADATA_FIELD_NUMBER: _ClassVar[int]
    TASK_RETRY_COUNT_FIELD_NUMBER: _ClassVar[int]
    task_run_external_id: str
    created_at: _timestamp_pb2.Timestamp
    message: str
    level: str
    metadata: str
    task_retry_count: int
    def __init__(self, task_run_external_id: _Optional[str] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., message: _Optional[str] = ..., level: _Optional[str] = ..., metadata: _Optional[str] = ..., task_retry_count: _Optional[int] = ...) -> None: ...

class PutLogResponse(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class PutStreamEventRequest(_message.Message):
    __slots__ = ("task_run_external_id", "created_at", "message", "metadata", "event_index")
    TASK_RUN_EXTERNAL_ID_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    METADATA_FIELD_NUMBER: _ClassVar[int]
    EVENT_INDEX_FIELD_NUMBER: _ClassVar[int]
    task_run_external_id: str
    created_at: _timestamp_pb2.Timestamp
    message: bytes
    metadata: str
    event_index: int
    def __init__(self, task_run_external_id: _Optional[str] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., message: _Optional[bytes] = ..., metadata: _Optional[str] = ..., event_index: _Optional[int] = ...) -> None: ...

class PutStreamEventResponse(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class BulkPushEventRequest(_message.Message):
    __slots__ = ("events",)
    EVENTS_FIELD_NUMBER: _ClassVar[int]
    events: _containers.RepeatedCompositeFieldContainer[PushEventRequest]
    def __init__(self, events: _Optional[_Iterable[_Union[PushEventRequest, _Mapping]]] = ...) -> None: ...

class PushEventRequest(_message.Message):
    __slots__ = ("key", "payload", "event_timestamp", "additional_metadata", "priority", "scope")
    KEY_FIELD_NUMBER: _ClassVar[int]
    PAYLOAD_FIELD_NUMBER: _ClassVar[int]
    EVENT_TIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    ADDITIONAL_METADATA_FIELD_NUMBER: _ClassVar[int]
    PRIORITY_FIELD_NUMBER: _ClassVar[int]
    SCOPE_FIELD_NUMBER: _ClassVar[int]
    key: str
    payload: str
    event_timestamp: _timestamp_pb2.Timestamp
    additional_metadata: str
    priority: int
    scope: str
    def __init__(self, key: _Optional[str] = ..., payload: _Optional[str] = ..., event_timestamp: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., additional_metadata: _Optional[str] = ..., priority: _Optional[int] = ..., scope: _Optional[str] = ...) -> None: ...

class ReplayEventRequest(_message.Message):
    __slots__ = ("event_id",)
    EVENT_ID_FIELD_NUMBER: _ClassVar[int]
    event_id: str
    def __init__(self, event_id: _Optional[str] = ...) -> None: ...
