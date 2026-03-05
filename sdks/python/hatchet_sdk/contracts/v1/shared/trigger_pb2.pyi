from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class WorkerLabelComparator(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    EQUAL: _ClassVar[WorkerLabelComparator]
    NOT_EQUAL: _ClassVar[WorkerLabelComparator]
    GREATER_THAN: _ClassVar[WorkerLabelComparator]
    GREATER_THAN_OR_EQUAL: _ClassVar[WorkerLabelComparator]
    LESS_THAN: _ClassVar[WorkerLabelComparator]
    LESS_THAN_OR_EQUAL: _ClassVar[WorkerLabelComparator]
EQUAL: WorkerLabelComparator
NOT_EQUAL: WorkerLabelComparator
GREATER_THAN: WorkerLabelComparator
GREATER_THAN_OR_EQUAL: WorkerLabelComparator
LESS_THAN: WorkerLabelComparator
LESS_THAN_OR_EQUAL: WorkerLabelComparator

class DesiredWorkerLabels(_message.Message):
    __slots__ = ("str_value", "int_value", "required", "comparator", "weight")
    STR_VALUE_FIELD_NUMBER: _ClassVar[int]
    INT_VALUE_FIELD_NUMBER: _ClassVar[int]
    REQUIRED_FIELD_NUMBER: _ClassVar[int]
    COMPARATOR_FIELD_NUMBER: _ClassVar[int]
    WEIGHT_FIELD_NUMBER: _ClassVar[int]
    str_value: str
    int_value: int
    required: bool
    comparator: WorkerLabelComparator
    weight: int
    def __init__(self, str_value: _Optional[str] = ..., int_value: _Optional[int] = ..., required: bool = ..., comparator: _Optional[_Union[WorkerLabelComparator, str]] = ..., weight: _Optional[int] = ...) -> None: ...

class TriggerWorkflowRequest(_message.Message):
    __slots__ = ("name", "input", "parent_id", "parent_task_run_external_id", "child_index", "child_key", "additional_metadata", "desired_worker_id", "priority", "desired_worker_labels")
    class DesiredWorkerLabelsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: DesiredWorkerLabels
        def __init__(self, key: _Optional[str] = ..., value: _Optional[_Union[DesiredWorkerLabels, _Mapping]] = ...) -> None: ...
    NAME_FIELD_NUMBER: _ClassVar[int]
    INPUT_FIELD_NUMBER: _ClassVar[int]
    PARENT_ID_FIELD_NUMBER: _ClassVar[int]
    PARENT_TASK_RUN_EXTERNAL_ID_FIELD_NUMBER: _ClassVar[int]
    CHILD_INDEX_FIELD_NUMBER: _ClassVar[int]
    CHILD_KEY_FIELD_NUMBER: _ClassVar[int]
    ADDITIONAL_METADATA_FIELD_NUMBER: _ClassVar[int]
    DESIRED_WORKER_ID_FIELD_NUMBER: _ClassVar[int]
    PRIORITY_FIELD_NUMBER: _ClassVar[int]
    DESIRED_WORKER_LABELS_FIELD_NUMBER: _ClassVar[int]
    name: str
    input: str
    parent_id: str
    parent_task_run_external_id: str
    child_index: int
    child_key: str
    additional_metadata: str
    desired_worker_id: str
    priority: int
    desired_worker_labels: _containers.MessageMap[str, DesiredWorkerLabels]
    def __init__(self, name: _Optional[str] = ..., input: _Optional[str] = ..., parent_id: _Optional[str] = ..., parent_task_run_external_id: _Optional[str] = ..., child_index: _Optional[int] = ..., child_key: _Optional[str] = ..., additional_metadata: _Optional[str] = ..., desired_worker_id: _Optional[str] = ..., priority: _Optional[int] = ..., desired_worker_labels: _Optional[_Mapping[str, DesiredWorkerLabels]] = ...) -> None: ...
