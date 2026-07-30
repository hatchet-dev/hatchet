from enum import Enum

from pydantic import BaseModel

from hatchet_sdk.contracts.dispatcher_pb2 import WorkerLabels
from hatchet_sdk.contracts.v1.shared.trigger_pb2 import DesiredWorkerLabels


class WorkerLabelComparator(int, Enum):
    EQUAL = 0
    NOT_EQUAL = 1
    GREATER_THAN = 2
    GREATER_THAN_OR_EQUAL = 3
    LESS_THAN = 4
    LESS_THAN_OR_EQUAL = 5


class WorkerLabel(BaseModel):
    key: str | None = None
    value: str | int

    def to_proto(self) -> WorkerLabels:
        if isinstance(self.value, int):
            return WorkerLabels(int_value=self.value)
        return WorkerLabels(str_value=str(self.value))


class DesiredWorkerLabel(WorkerLabel):
    required: bool = False
    weight: int | None = None
    comparator: int | WorkerLabelComparator | None = None

    def to_proto(self) -> DesiredWorkerLabels:  # type: ignore[override]
        return DesiredWorkerLabels(
            str_value=self.value if not isinstance(self.value, int) else None,
            int_value=self.value if isinstance(self.value, int) else None,
            required=self.required,
            weight=self.weight,
            comparator=self.comparator,  # type: ignore[arg-type]
        )
