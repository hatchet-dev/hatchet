from datetime import UTC, datetime
from zoneinfo import ZoneInfo

from hatchet_sdk.utils.datetimes import partition_date_range


def test_partition_date_ranges_single_day() -> None:
    since = datetime(2025, 1, 1, 1, 2, 3, tzinfo=UTC)
    until = datetime(2025, 1, 1, 3, 14, 15, tzinfo=UTC)
    partitioned = partition_date_range(
        since,
        until,
    )

    assert len(partitioned) == 1
    assert partitioned[0] == (since, until)


def test_partition_date_ranges_multi_day() -> None:
    since = datetime(2025, 1, 1, 1, 2, 3, tzinfo=UTC)
    until = datetime(2025, 1, 4, 3, 14, 15, tzinfo=UTC)
    partitioned = partition_date_range(
        since,
        until,
    )

    assert len(partitioned) == 4
    assert partitioned[0] == (
        since,
        datetime(2025, 1, 2, tzinfo=UTC),
    )
    assert partitioned[1] == (
        datetime(2025, 1, 2, tzinfo=UTC),
        datetime(2025, 1, 3, tzinfo=UTC),
    )
    assert partitioned[2] == (
        datetime(2025, 1, 3, tzinfo=UTC),
        datetime(2025, 1, 4, tzinfo=UTC),
    )
    assert partitioned[3] == (
        datetime(2025, 1, 4, tzinfo=UTC),
        until,
    )


def test_partition_date_ranges_non_utc() -> None:
    since = datetime(2025, 1, 1, 22, 2, 3, tzinfo=ZoneInfo("America/New_York"))
    until = datetime(2025, 1, 4, 3, 14, 15, tzinfo=UTC)

    partitioned = partition_date_range(
        since,
        until,
    )

    assert len(partitioned) == 3
    assert partitioned[0] == (
        since.astimezone(UTC),
        datetime(2025, 1, 3, tzinfo=UTC),
    )
    assert partitioned[1] == (
        datetime(2025, 1, 3, tzinfo=UTC),
        datetime(2025, 1, 4, tzinfo=UTC),
    )
    assert partitioned[2] == (
        datetime(2025, 1, 4, tzinfo=UTC),
        until,
    )
