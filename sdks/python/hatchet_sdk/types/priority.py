from typing import Literal

from hatchet_sdk.utils.literals import convert_literal_using_mapping

Priority = Literal["LOW", "MEDIUM", "HIGH"]

_PRIORITY_TO_INT: dict[Priority, int] = {"LOW": 1, "MEDIUM": 2, "HIGH": 3}
_INT_TO_PRIORITY: dict[int, Priority] = {
    value: priority for priority, value in _PRIORITY_TO_INT.items()
}


def priority_to_int(priority: Priority) -> int:
    return convert_literal_using_mapping(priority, _PRIORITY_TO_INT, "priority")


def priority_from_int(value: int) -> Priority:
    if value not in _INT_TO_PRIORITY:
        raise ValueError(f"Priority must be 1, 2, or 3. Got: {value}")

    return _INT_TO_PRIORITY[value]
