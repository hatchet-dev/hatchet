from typing import Literal

from pydantic import BaseModel

from hatchet_sdk.contracts.dispatcher_pb2 import WorkerLabels
from hatchet_sdk.contracts.v1.shared.trigger_pb2 import DesiredWorkerLabels
from hatchet_sdk.contracts.v1.shared.trigger_pb2 import (
    WorkerLabelComparator as WorkerLabelComparatorProto,
)
from hatchet_sdk.utils.proto_enums import convert_python_literal_to_proto

WorkerLabelComparator = Literal[
    "EQUAL",
    "NOT_EQUAL",
    "GREATER_THAN",
    "GREATER_THAN_OR_EQUAL",
    "LESS_THAN",
    "LESS_THAN_OR_EQUAL",
]


class WorkerLabel(BaseModel):
    key: str
    value: str | int

    def to_proto(self) -> WorkerLabels:
        if isinstance(self.value, int):
            return WorkerLabels(int_value=self.value)
        return WorkerLabels(str_value=str(self.value))


class DesiredWorkerLabel(BaseModel):
    key: str
    value: str | int
    required: bool = False
    weight: int | None = None
    comparator: WorkerLabelComparator | None = None

    def to_proto(self) -> DesiredWorkerLabels:
        return DesiredWorkerLabels(
            str_value=self.value if not isinstance(self.value, int) else None,
            int_value=self.value if isinstance(self.value, int) else None,
            required=self.required,
            weight=self.weight,
            comparator=convert_python_literal_to_proto(
                self.comparator, WorkerLabelComparatorProto
            ),
        )
