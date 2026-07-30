import asyncio
from datetime import datetime

import tenacity
from tenacity import stop_after_attempt, wait_exponential, wait_fixed

from hatchet_sdk import Hatchet, RunStatus
from hatchet_sdk.clients.rest.models.v1_event import V1Event
from hatchet_sdk.clients.rest.models.v1_task_status import V1TaskStatus
from hatchet_sdk.clients.rest.models.v1_task_summary import V1TaskSummary


async def wait_for_running_status(
    hatchet: Hatchet, run_id: str, timeout: float = 60.0
) -> None:
    interval = 0.5
    max_iters = int(timeout / interval)
    for _ in range(max_iters):
        run = await hatchet.runs.aio_get_details(run_id)
        if run.status == RunStatus.RUNNING:
            return
        await asyncio.sleep(interval)


async def poll_for_runs(
    hatchet: Hatchet,
    *,
    expected_count: int,
    additional_metadata: dict[str, str] | None = None,
    workflow_ids: list[str] | None = None,
    parent_task_external_id: str | None = None,
    triggering_event_external_id: str | None = None,
    statuses: list[V1TaskStatus] | None = None,
    timeout: float = 60.0,
    interval: float = 0.5,
) -> list[V1TaskSummary]:
    terminal = statuses or [
        V1TaskStatus.COMPLETED,
        V1TaskStatus.FAILED,
        V1TaskStatus.CANCELLED,
    ]
    max_iters = int(timeout / interval)
    runs: list[V1TaskSummary] = []
    for _ in range(max_iters):
        runs = await hatchet.runs.aio_list(
            additional_metadata=additional_metadata,
            workflow_ids=workflow_ids,
            parent_task_external_id=parent_task_external_id,
            triggering_event_external_id=triggering_event_external_id,
            limit=1000,
        )
        if len(runs) >= expected_count and all(r.status in terminal for r in runs):
            return runs
        await asyncio.sleep(interval)
    return runs


async def wait_for_replay(hatchet: Hatchet, run_id: str, timeout: float = 30.0) -> None:
    interval = 0.25
    max_iters = int(timeout / interval)
    for _ in range(max_iters):
        run = await hatchet.runs.aio_get_details(run_id)
        if run.status != RunStatus.COMPLETED:
            return
        await asyncio.sleep(interval)


async def wait_for_event(
    hatchet: Hatchet,
    webhook_name: str,
    test_start: datetime,
) -> V1Event | None:
    @tenacity.retry(stop=stop_after_attempt(10), wait=wait_fixed(3))
    async def get_events() -> V1Event | None:
        events = await hatchet.event.aio_list(since=test_start)
        if not events:
            raise Exception()
        filtered_event = next(
            (
                event
                for event in events
                if event.triggering_webhook_name == webhook_name
            ),
            None,
        )
        if not filtered_event:
            raise Exception()
        return filtered_event

    try:
        return await get_events()
    except tenacity.RetryError:
        return None


async def wait_for_workflow_run(
    hatchet: Hatchet, event_id: str, test_start: datetime
) -> V1TaskSummary | None:
    @tenacity.retry(
        stop=stop_after_attempt(5), wait=wait_exponential(multiplier=1, min=4, max=10)
    )
    async def get_runs() -> V1TaskSummary:
        runs = await hatchet.runs.aio_list(
            since=test_start,
            additional_metadata={
                "hatchet__event_id": event_id,
            },
        )
        for row in runs:
            if row.status == V1TaskStatus.COMPLETED:
                return row
        raise Exception()

    return await get_runs()
