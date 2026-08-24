import asyncio
from datetime import datetime

import tenacity
from tenacity import stop_after_attempt, wait_exponential

from hatchet_sdk import Hatchet, RunStatus
from hatchet_sdk.clients.rest.models.v1_event import V1Event
from hatchet_sdk.clients.rest.models.v1_task_summary import V1TaskSummary
from hatchet_sdk.clients.rest.models.v1_task_status import V1TaskStatus


async def wait_for_running_status(
    hatchet: Hatchet,
    run_id: str,
    timeout: float = 60.0,
    min_task_runs: int = 0,
) -> None:
    """Poll until the workflow run reaches RUNNING status (and has at least min_task_runs child tasks) or timeout is exceeded."""
    interval = 0.5
    max_iters = int(timeout / interval)
    for _ in range(max_iters):
        run = await hatchet.runs.aio_get_details(run_id)
        if run.status == RunStatus.RUNNING and len(run.task_runs) >= min_task_runs:
            return
        await asyncio.sleep(interval)


async def wait_for_event(
    hatchet: Hatchet,
    webhook_name: str,
    test_start: datetime,
) -> V1Event | None:
    await asyncio.sleep(5)

    @tenacity.retry(
        stop=stop_after_attempt(5), wait=wait_exponential(multiplier=1, min=4, max=10)
    )
    async def get_events() -> V1Event | None:
        events = await hatchet.event.aio_list(since=test_start)
        if not events.rows:
            raise Exception()
        filtered_event = next(
            (
                event
                for event in events.rows
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
        for row in runs.rows:
            if row.status == V1TaskStatus.COMPLETED:
                return row
        raise Exception()

    return await get_runs()
