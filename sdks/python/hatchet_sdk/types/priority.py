import warnings
from enum import Enum


class Priority(int, Enum):
    LOW = 1
    MEDIUM = 2
    HIGH = 3


def _warn_if_int_priority(
    *priorities: "Priority | int | None", stacklevel: int = 3
) -> None:
    if any(p is not None and not isinstance(p, Priority) for p in priorities):
        warnings.warn(
            "Passing priority as an int is deprecated and will be removed in v2.0.0. Use Priority enum values (Priority.LOW, Priority.MEDIUM, Priority.HIGH) instead.",
            DeprecationWarning,
            stacklevel=stacklevel,
        )
