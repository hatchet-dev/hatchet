from typing import Literal, cast, get_args

RunStatus = Literal["QUEUED", "RUNNING", "COMPLETED", "CANCELLED", "FAILED"]

_RUN_STATUSES: frozenset[str] = frozenset(get_args(RunStatus))


def parse_run_status(value: str) -> RunStatus:
    if value not in _RUN_STATUSES:
        raise ValueError(f"Unknown run status: {value}")

    return cast("RunStatus", value)
