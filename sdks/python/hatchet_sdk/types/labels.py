import warnings
from enum import Enum

from pydantic import BaseModel, field_validator

from hatchet_sdk.contracts.v1.workflows_pb2 import DesiredWorkerLabels


class WorkerLabelComparator(int, Enum):
    EQUAL = 0
    NOT_EQUAL = 1
    GREATER_THAN = 2
    GREATER_THAN_OR_EQUAL = 3
    LESS_THAN = 4
    LESS_THAN_OR_EQUAL = 5


def _warn_if_int_comparator(
    *comparators: "WorkerLabelComparator | int | None", stacklevel: int = 3
) -> None:
    if any(
        c is not None and not isinstance(c, WorkerLabelComparator) for c in comparators
    ):
        warnings.warn(
            "Passing comparator as an int is deprecated and will be removed in v2.0.0. Use WorkerLabelComparator enum values instead.",
            DeprecationWarning,
            stacklevel=stacklevel,
        )


class WorkerLabel(BaseModel):
    key: str | None = None
    value: str | int


class DesiredWorkerLabel(WorkerLabel):
    required: bool = False
    weight: int | None = None
    comparator: int | WorkerLabelComparator | None = None

    @field_validator("comparator", mode="before")
    @classmethod
    def _check_comparator_type(cls, v: object) -> object:
        _warn_if_int_comparator(v, stacklevel=5)  # type: ignore[arg-type]
        return v

    def to_proto(self) -> DesiredWorkerLabels:
        return DesiredWorkerLabels(
            str_value=self.value if not isinstance(self.value, int) else None,
            int_value=self.value if isinstance(self.value, int) else None,
            required=self.required,
            weight=self.weight,
            comparator=self.comparator,  # type: ignore[arg-type]
        )
