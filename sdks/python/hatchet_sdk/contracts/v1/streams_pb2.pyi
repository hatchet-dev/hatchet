import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class PublishStreamMessageRequest(_message.Message):
    __slots__ = ("namespace", "topic", "payload", "producer_id", "producer_seq", "payload_ref")
    NAMESPACE_FIELD_NUMBER: _ClassVar[int]
    TOPIC_FIELD_NUMBER: _ClassVar[int]
    PAYLOAD_FIELD_NUMBER: _ClassVar[int]
    PRODUCER_ID_FIELD_NUMBER: _ClassVar[int]
    PRODUCER_SEQ_FIELD_NUMBER: _ClassVar[int]
    PAYLOAD_REF_FIELD_NUMBER: _ClassVar[int]
    namespace: str
    topic: str
    payload: bytes
    producer_id: str
    producer_seq: int
    payload_ref: str
    def __init__(self, namespace: _Optional[str] = ..., topic: _Optional[str] = ..., payload: _Optional[bytes] = ..., producer_id: _Optional[str] = ..., producer_seq: _Optional[int] = ..., payload_ref: _Optional[str] = ...) -> None: ...

class PublishStreamMessageResponse(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class PublishStreamMessagesRequest(_message.Message):
    __slots__ = ("namespace", "topic", "producer_id", "first_producer_seq", "entries")
    NAMESPACE_FIELD_NUMBER: _ClassVar[int]
    TOPIC_FIELD_NUMBER: _ClassVar[int]
    PRODUCER_ID_FIELD_NUMBER: _ClassVar[int]
    FIRST_PRODUCER_SEQ_FIELD_NUMBER: _ClassVar[int]
    ENTRIES_FIELD_NUMBER: _ClassVar[int]
    namespace: str
    topic: str
    producer_id: str
    first_producer_seq: int
    entries: _containers.RepeatedCompositeFieldContainer[StreamPublishEntry]
    def __init__(self, namespace: _Optional[str] = ..., topic: _Optional[str] = ..., producer_id: _Optional[str] = ..., first_producer_seq: _Optional[int] = ..., entries: _Optional[_Iterable[_Union[StreamPublishEntry, _Mapping]]] = ...) -> None: ...

class StreamPublishEntry(_message.Message):
    __slots__ = ("payload", "payload_ref")
    PAYLOAD_FIELD_NUMBER: _ClassVar[int]
    PAYLOAD_REF_FIELD_NUMBER: _ClassVar[int]
    payload: bytes
    payload_ref: str
    def __init__(self, payload: _Optional[bytes] = ..., payload_ref: _Optional[str] = ...) -> None: ...

class PublishStreamMessagesResponse(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class SubscribeStreamRequest(_message.Message):
    __slots__ = ("namespace", "topic", "cursor")
    NAMESPACE_FIELD_NUMBER: _ClassVar[int]
    TOPIC_FIELD_NUMBER: _ClassVar[int]
    CURSOR_FIELD_NUMBER: _ClassVar[int]
    namespace: str
    topic: str
    cursor: str
    def __init__(self, namespace: _Optional[str] = ..., topic: _Optional[str] = ..., cursor: _Optional[str] = ...) -> None: ...

class StreamMessage(_message.Message):
    __slots__ = ("entries", "hangup", "cursor")
    ENTRIES_FIELD_NUMBER: _ClassVar[int]
    HANGUP_FIELD_NUMBER: _ClassVar[int]
    CURSOR_FIELD_NUMBER: _ClassVar[int]
    entries: _containers.RepeatedCompositeFieldContainer[StreamEntry]
    hangup: bool
    cursor: str
    def __init__(self, entries: _Optional[_Iterable[_Union[StreamEntry, _Mapping]]] = ..., hangup: bool = ..., cursor: _Optional[str] = ...) -> None: ...

class StreamEntry(_message.Message):
    __slots__ = ("payload", "cursor", "created_at", "payload_ref")
    PAYLOAD_FIELD_NUMBER: _ClassVar[int]
    CURSOR_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    PAYLOAD_REF_FIELD_NUMBER: _ClassVar[int]
    payload: bytes
    cursor: str
    created_at: _timestamp_pb2.Timestamp
    payload_ref: str
    def __init__(self, payload: _Optional[bytes] = ..., cursor: _Optional[str] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., payload_ref: _Optional[str] = ...) -> None: ...

class GetStreamTopicMetadataRequest(_message.Message):
    __slots__ = ("namespace", "topic")
    NAMESPACE_FIELD_NUMBER: _ClassVar[int]
    TOPIC_FIELD_NUMBER: _ClassVar[int]
    namespace: str
    topic: str
    def __init__(self, namespace: _Optional[str] = ..., topic: _Optional[str] = ...) -> None: ...

class StreamTopicMetadata(_message.Message):
    __slots__ = ("namespace", "topic", "tenant_id", "latest_cursor", "message_count", "last_published_at")
    NAMESPACE_FIELD_NUMBER: _ClassVar[int]
    TOPIC_FIELD_NUMBER: _ClassVar[int]
    TENANT_ID_FIELD_NUMBER: _ClassVar[int]
    LATEST_CURSOR_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_COUNT_FIELD_NUMBER: _ClassVar[int]
    LAST_PUBLISHED_AT_FIELD_NUMBER: _ClassVar[int]
    namespace: str
    topic: str
    tenant_id: str
    latest_cursor: str
    message_count: int
    last_published_at: _timestamp_pb2.Timestamp
    def __init__(self, namespace: _Optional[str] = ..., topic: _Optional[str] = ..., tenant_id: _Optional[str] = ..., latest_cursor: _Optional[str] = ..., message_count: _Optional[int] = ..., last_published_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class ResetStreamTopicRequest(_message.Message):
    __slots__ = ("namespace", "topic", "cursor")
    NAMESPACE_FIELD_NUMBER: _ClassVar[int]
    TOPIC_FIELD_NUMBER: _ClassVar[int]
    CURSOR_FIELD_NUMBER: _ClassVar[int]
    namespace: str
    topic: str
    cursor: str
    def __init__(self, namespace: _Optional[str] = ..., topic: _Optional[str] = ..., cursor: _Optional[str] = ...) -> None: ...

class ResetStreamTopicResponse(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...
