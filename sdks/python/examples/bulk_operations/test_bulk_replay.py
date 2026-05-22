import asyncio
from datetime import datetime, timedelta, timezone
from uuid import uuid4

import pytest
import tenacity
from tenacity import stop_after_attempt, wait_exponential

from examples.bulk_operations.worker import (
    bulk_replay_test_1,
    bulk_replay_test_2,
    bulk_replay_test_3,
)
from hatchet_sdk import BulkCancelReplayOpts, Hatchet, RunFilter
from hatchet_sdk.clients.rest.models.v1_task_summary_list import V1TaskSummaryList
from hatchet_sdk.clients.rest.models.v1_task_summary import V1TaskSummary
from hatchet_sdk.clients.rest.models.v1_task_status import V1TaskStatus


@pytest.mark.asyncio(loop_scope="session")
async def test_bulk_replay(hatchet: Hatchet) -> None:
    test_run_id = str(uuid4())
    n = 100
    test_start = datetime.now(tz=timezone.utc)
    with pytest.raises(Exception):
        await bulk_replay_test_1.aio_run_many(
            [
                bulk_replay_test_1.create_bulk_run_item(
                    additional_metadata={
                        "test_run_id": test_run_id,
                    },
                )
                for _ in range(n + 1)
            ]
        )

    with pytest.raises(Exception):
        await bulk_replay_test_2.aio_run_many(
            [
                bulk_replay_test_2.create_bulk_run_item(
                    additional_metadata={
                        "test_run_id": test_run_id,
                    },
                )
                for _ in range((n // 2) - 1)
            ]
        )

    with pytest.raises(Exception):
        await bulk_replay_test_3.aio_run_many(
            [
                bulk_replay_test_3.create_bulk_run_item(
                    additional_metadata={
                        "test_run_id": test_run_id,
                    },
                )
                for _ in range((n // 2) - 2)
            ]
        )

    workflow_ids = [
        bulk_replay_test_1.id,
        bulk_replay_test_2.id,
        bulk_replay_test_3.id,
    ]

    ## Should result in two batches of replays
    await hatchet.runs.aio_bulk_replay(
        opts=BulkCancelReplayOpts(
            filters=RunFilter(
                workflow_ids=workflow_ids,
                since=test_start,
                additional_metadata={"test_run_id": test_run_id},
            )
        )
    )

    await asyncio.sleep(20)

    @tenacity.retry(
        stop=stop_after_attempt(10), wait=wait_exponential(multiplier=1, min=4, max=10)
    )
    async def get_runs() -> list[V1TaskSummary]:
        runs = await hatchet.runs.aio_list(
            workflow_ids=workflow_ids,
            since=test_start,
            additional_metadata={"test_run_id": test_run_id},
            limit=1000,
        )

        def predicate(r: V1TaskSummary) -> bool:
            return (
                r.status == V1TaskStatus.COMPLETED  # type: ignore[return-value]
                and r.retry_count
                and r.retry_count >= 1
                and r.attempt
                and r.attempt >= 2
            )

        for r in runs:
            if not predicate(r):
                raise Exception
        return runs

    runs = await get_runs()

    assert len(runs) == n + 1 + (n // 2 - 1) + (n // 2 - 2)

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
