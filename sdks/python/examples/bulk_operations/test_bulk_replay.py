from datetime import datetime, timezone

import pytest
import tenacity
from tenacity import stop_after_attempt, wait_exponential

from examples.bulk_operations.worker import (
    bulk_replay_test_1,
    bulk_replay_test_2,
    bulk_replay_test_3,
)
from hatchet_sdk import BulkCancelReplayOpts, Hatchet, RunFilter
from hatchet_sdk.clients.rest.models.v1_task_status import V1TaskStatus
from hatchet_sdk.clients.rest.models.v1_task_summary import V1TaskSummary


@pytest.mark.asyncio(loop_scope="session")
async def test_bulk_replay(hatchet: Hatchet, test_run_id: str) -> None:
    n = 100
    expected_total = n + 1 + (n // 2 - 1) + (n // 2 - 2)
    test_start = datetime.now(tz=timezone.utc)

    workflow_ids = [
        bulk_replay_test_1.id,
        bulk_replay_test_2.id,
        bulk_replay_test_3.id,
    ]
    additional_metadata = {"test_run_id": test_run_id}

    async def list_filtered_runs() -> list[V1TaskSummary]:
        return await hatchet.runs.aio_list(
            workflow_ids=workflow_ids,
            since=test_start,
            additional_metadata=additional_metadata,
            limit=1000,
        )

    def summarize_statuses(rows: list[V1TaskSummary]) -> str:
        counts: dict[str, int] = {}
        for row in rows:
            status = str(row.status)
            counts[status] = counts.get(status, 0) + 1
        return ", ".join(
            f"{status}={count}" for status, count in sorted(counts.items())
        )

    await bulk_replay_test_1.aio_run_many(
        [
            bulk_replay_test_1.create_bulk_run_item(
                additional_metadata=additional_metadata,
            )
            for _ in range(n + 1)
        ],
        wait_for_result=False,
    )

    await bulk_replay_test_2.aio_run_many(
        [
            bulk_replay_test_2.create_bulk_run_item(
                additional_metadata=additional_metadata,
            )
            for _ in range((n // 2) - 1)
        ],
        wait_for_result=False,
    )

    await bulk_replay_test_3.aio_run_many(
        [
            bulk_replay_test_3.create_bulk_run_item(
                additional_metadata=additional_metadata,
            )
            for _ in range((n // 2) - 2)
        ],
        wait_for_result=False,
    )

    @tenacity.retry(
        stop=stop_after_attempt(10), wait=wait_exponential(multiplier=1, min=4, max=10)
    )
    async def wait_for_all_failed() -> list[V1TaskSummary]:
        rows = await list_filtered_runs()

        if len(rows) != expected_total:
            raise AssertionError(
                f"Expected {expected_total} runs before replay, got {len(rows)} "
                f"(statuses: {summarize_statuses(rows)})"
            )

        # Every run fails on its first attempt, but the engine may internally
        # retry a run whose assignment could not be delivered (worker briefly
        # unreachable or a lost heartbeat under load), and internal retries
        # bump retry_count. Such a run then completes with retry_count >= 1
        # before we ever replay it. Treat that as settled: the precondition
        # that matters is that no run completed on its first attempt.
        settled_count = sum(
            1
            for row in rows
            if row.status == V1TaskStatus.FAILED
            or (
                row.status == V1TaskStatus.COMPLETED
                and row.retry_count is not None
                and row.retry_count >= 1
            )
        )
        if settled_count != expected_total:
            raise AssertionError(
                f"Expected all {expected_total} runs to be FAILED (or COMPLETED "
                f"only via an engine internal retry) before replay, but "
                f"{settled_count}/{expected_total} are "
                f"(statuses: {summarize_statuses(rows)})"
            )

        first_attempt_completed = [
            row
            for row in rows
            if row.status == V1TaskStatus.COMPLETED and not row.retry_count
        ]
        assert (
            not first_attempt_completed
        ), f"{len(first_attempt_completed)} runs completed on their first attempt"

        return runs

    before_replay = await wait_for_all_failed()

    # A run that an internal retry already completed satisfies the post-replay
    # status checks on its own, so remember every run's attempt count and
    # require the replay to have moved each one forward.
    attempts_before_replay = {
        row.metadata.id: row.attempt or 0 for row in before_replay
    }

    await hatchet.runs.aio_bulk_replay(
        opts=BulkCancelReplayOpts(
            filters=RunFilter(
                workflow_ids=workflow_ids,
                since=test_start,
                additional_metadata=additional_metadata,
            )
        )
    )

    @tenacity.retry(
        stop=stop_after_attempt(10), wait=wait_exponential(multiplier=1, min=4, max=10)
    )
    async def wait_for_replayed_completed() -> list[V1TaskSummary]:
        rows = await list_filtered_runs()

        if len(rows) != expected_total:
            raise AssertionError(
                f"Expected {expected_total} runs after replay, got {len(rows)} "
                f"(statuses: {summarize_statuses(rows)})"
            )

        ready_count = sum(
            1
            for row in rows
            if row.status == V1TaskStatus.COMPLETED
            and row.retry_count is not None
            and row.retry_count >= 1
            and row.attempt is not None
            and row.attempt >= 2
        )
        if ready_count != expected_total:
            raise AssertionError(
                f"Expected all {expected_total} runs to be COMPLETED with "
                f"retry_count>=1 and attempt>=2 after replay, but only "
                f"{ready_count}/{expected_total} are ready "
                f"(statuses: {summarize_statuses(rows)})"
            )

        return rows

    runs = await wait_for_replayed_completed()

    assert len(runs) == expected_total

    not_replayed = [
        row.metadata.id
        for row in runs
        if (row.attempt or 0) <= attempts_before_replay.get(row.metadata.id, 0)
    ]
    assert (
        not not_replayed
    ), f"{len(not_replayed)} runs were not re-run by the bulk replay: {not_replayed}"

    for run in runs:
        assert run.status == V1TaskStatus.COMPLETED
        assert run.retry_count and run.retry_count >= 1
        assert run.attempt and run.attempt >= 2

    assert len([r for r in runs if r.workflow_id == bulk_replay_test_1.id]) == n + 1
    assert (
        len([r for r in runs if r.workflow_id == bulk_replay_test_2.id]) == n // 2 - 1
    )
    assert (
        len([r for r in runs if r.workflow_id == bulk_replay_test_3.id]) == n // 2 - 2
    )
