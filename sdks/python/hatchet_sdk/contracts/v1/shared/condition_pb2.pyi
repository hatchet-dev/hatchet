from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from typing import ClassVar as _ClassVar, Iterable as _Iterable, Mapping as _Mapping, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Action(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    CREATE: _ClassVar[Action]
    QUEUE: _ClassVar[Action]
    CANCEL: _ClassVar[Action]
    SKIP: _ClassVar[Action]
CREATE: Action
QUEUE: Action
CANCEL: Action
SKIP: Action

class BaseMatchCondition(_message.Message):
    __slots__ = ("readable_data_key", "action", "or_group_id", "expression")
    READABLE_DATA_KEY_FIELD_NUMBER: _ClassVar[int]
    ACTION_FIELD_NUMBER: _ClassVar[int]
    OR_GROUP_ID_FIELD_NUMBER: _ClassVar[int]
    EXPRESSION_FIELD_NUMBER: _ClassVar[int]
    readable_data_key: str
    action: Action
    or_group_id: str
    expression: str
    def __init__(self, readable_data_key: _Optional[str] = ..., action: _Optional[_Union[Action, str]] = ..., or_group_id: _Optional[str] = ..., expression: _Optional[str] = ...) -> None: ...

class ParentOverrideMatchCondition(_message.Message):
    __slots__ = ("base", "parent_readable_id")
    BASE_FIELD_NUMBER: _ClassVar[int]
    PARENT_READABLE_ID_FIELD_NUMBER: _ClassVar[int]
    base: BaseMatchCondition
    parent_readable_id: str
    def __init__(self, base: _Optional[_Union[BaseMatchCondition, _Mapping]] = ..., parent_readable_id: _Optional[str] = ...) -> None: ...

class SleepMatchCondition(_message.Message):
    __slots__ = ("base", "sleep_for")
    BASE_FIELD_NUMBER: _ClassVar[int]
    SLEEP_FOR_FIELD_NUMBER: _ClassVar[int]
    base: BaseMatchCondition
    sleep_for: str
    def __init__(self, base: _Optional[_Union[BaseMatchCondition, _Mapping]] = ..., sleep_for: _Optional[str] = ...) -> None: ...

class UserEventMatchCondition(_message.Message):
    __slots__ = ("base", "user_event_key")
    BASE_FIELD_NUMBER: _ClassVar[int]
    USER_EVENT_KEY_FIELD_NUMBER: _ClassVar[int]
    base: BaseMatchCondition
    user_event_key: str
    def __init__(self, base: _Optional[_Union[BaseMatchCondition, _Mapping]] = ..., user_event_key: _Optional[str] = ...) -> None: ...

class TaskConditions(_message.Message):
    __slots__ = ("parent_override_conditions", "sleep_conditions", "user_event_conditions")
    PARENT_OVERRIDE_CONDITIONS_FIELD_NUMBER: _ClassVar[int]
    SLEEP_CONDITIONS_FIELD_NUMBER: _ClassVar[int]
    USER_EVENT_CONDITIONS_FIELD_NUMBER: _ClassVar[int]
    parent_override_conditions: _containers.RepeatedCompositeFieldContainer[ParentOverrideMatchCondition]
    sleep_conditions: _containers.RepeatedCompositeFieldContainer[SleepMatchCondition]
    user_event_conditions: _containers.RepeatedCompositeFieldContainer[UserEventMatchCondition]
    def __init__(self, parent_override_conditions: _Optional[_Iterable[_Union[ParentOverrideMatchCondition, _Mapping]]] = ..., sleep_conditions: _Optional[_Iterable[_Union[SleepMatchCondition, _Mapping]]] = ..., user_event_conditions: _Optional[_Iterable[_Union[UserEventMatchCondition, _Mapping]]] = ...) -> None: ...

class DurableEventListenerConditions(_message.Message):
    __slots__ = ("sleep_conditions", "user_event_conditions")
    SLEEP_CONDITIONS_FIELD_NUMBER: _ClassVar[int]
    USER_EVENT_CONDITIONS_FIELD_NUMBER: _ClassVar[int]
    sleep_conditions: _containers.RepeatedCompositeFieldContainer[SleepMatchCondition]
    user_event_conditions: _containers.RepeatedCompositeFieldContainer[UserEventMatchCondition]
    def __init__(self, sleep_conditions: _Optional[_Iterable[_Union[SleepMatchCondition, _Mapping]]] = ..., user_event_conditions: _Optional[_Iterable[_Union[UserEventMatchCondition, _Mapping]]] = ...) -> None: ...
