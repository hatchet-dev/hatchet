from datetime import datetime
from typing import Mapping, Sequence, Type

from pydantic import BaseModel


class WorkflowValidator(BaseModel):
    workflow_input: Type[BaseModel] | None = None
    step_output: Type[BaseModel] | None = None


JSONSerializableMapping = Mapping[
    str,
    str
    | int
    | float
    | bool
    | datetime
    | None
    | Sequence[str]
    | Sequence[int]
    | Sequence[float]
    | Sequence[bool]
    | Sequence[datetime]
    | Sequence[None],
]
