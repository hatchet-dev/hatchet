from datetime import datetime, timedelta, timezone
from typing import TypeVar

T = TypeVar("T")
R = TypeVar("R")


def _to_utc(dt: datetime) -> datetime:
    if not dt.tzinfo:
        return dt.replace(tzinfo=timezone.utc)

    return dt.astimezone(timezone.utc)


def partition_date_range(
    since: datetime, until: datetime
) -> list[tuple[datetime, datetime]]:
    since = _to_utc(since)
    until = _to_utc(until)

    ranges = []
    current_start = since

    while current_start < until:
        next_day = (current_start + timedelta(days=1)).replace(
            hour=0, minute=0, second=0, microsecond=0
        )

        current_end = min(next_day, until)

        ranges.append((current_start, current_end))

        current_start = next_day

    return ranges
