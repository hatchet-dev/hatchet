from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from typing import ClassVar as _ClassVar, Optional as _Optional

DESCRIPTOR: _descriptor.FileDescriptor

class TriggerWorkflowRequest(_message.Message):
    __slots__ = ("name", "input", "parent_id", "parent_task_run_external_id", "child_index", "child_key", "additional_metadata", "desired_worker_id", "priority")
    NAME_FIELD_NUMBER: _ClassVar[int]
    INPUT_FIELD_NUMBER: _ClassVar[int]
    PARENT_ID_FIELD_NUMBER: _ClassVar[int]
    PARENT_TASK_RUN_EXTERNAL_ID_FIELD_NUMBER: _ClassVar[int]
    CHILD_INDEX_FIELD_NUMBER: _ClassVar[int]
    CHILD_KEY_FIELD_NUMBER: _ClassVar[int]
    ADDITIONAL_METADATA_FIELD_NUMBER: _ClassVar[int]
    DESIRED_WORKER_ID_FIELD_NUMBER: _ClassVar[int]
    PRIORITY_FIELD_NUMBER: _ClassVar[int]
    name: str
    input: str
    parent_id: str
    parent_task_run_external_id: str
    child_index: int
    child_key: str
    additional_metadata: str
    desired_worker_id: str
    priority: int
    def __init__(self, name: _Optional[str] = ..., input: _Optional[str] = ..., parent_id: _Optional[str] = ..., parent_task_run_external_id: _Optional[str] = ..., child_index: _Optional[int] = ..., child_key: _Optional[str] = ..., additional_metadata: _Optional[str] = ..., desired_worker_id: _Optional[str] = ..., priority: _Optional[int] = ...) -> None: ...
