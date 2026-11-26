import asyncio
from typing import Any

import pytest

from examples.bug_tests.root_task_conditions.worker import (
    ROOT_SKIP_IF_EVENT_KEY,
    ROOT_WAIT_FOR_EVENT_KEY,
    root_cancel_if_sleep,
    root_skip_if_event,
    root_skip_if_event_task,
    root_wait_for_event,
    root_wait_for_event_task,
    root_wait_for_sleep,
    root_wait_for_sleep_task,
)
from hatchet_sdk import CancelledRunError, Hatchet
from hatchet_sdk.runnables.workflow import Workflow


async def push_event_until_run_finishes(
    hatchet: Hatchet, workflow: Workflow[Any], event_key: str
) -> dict[str, Any]:
    ref = await workflow.aio_run(wait_for_result=False)
    result = asyncio.create_task(ref.aio_result())

    # An event pushed before the engine registers the match is dropped silently,
    # so keep pushing until the run finishes.
    while not result.done():
        await hatchet.events.aio_push(event_key, {})
        await asyncio.sleep(1)

    return await result


@pytest.mark.timeout(30, func_only=True)
@pytest.mark.asyncio(loop_scope="session")
async def test_single_root_task_runs_after_wait_for_sleep() -> None:
    result = await root_wait_for_sleep.aio_run()

    assert result[root_wait_for_sleep_task.name] == {"ran": True}


@pytest.mark.timeout(30, func_only=True)
@pytest.mark.asyncio(loop_scope="session")
async def test_single_root_task_runs_after_wait_for_event(hatchet: Hatchet) -> None:
    result = await push_event_until_run_finishes(
        hatchet, root_wait_for_event, ROOT_WAIT_FOR_EVENT_KEY
    )

    assert result[root_wait_for_event_task.name] == {"ran": True}


@pytest.mark.timeout(30, func_only=True)
@pytest.mark.asyncio(loop_scope="session")
async def test_single_root_task_is_skipped_on_skip_if_event(hatchet: Hatchet) -> None:
    result = await push_event_until_run_finishes(
        hatchet, root_skip_if_event, ROOT_SKIP_IF_EVENT_KEY
    )

    assert result[root_skip_if_event_task.name] == {"skipped": True}


@pytest.mark.timeout(30, func_only=True)
@pytest.mark.asyncio(loop_scope="session")
async def test_single_root_task_is_cancelled_on_cancel_if_sleep(
    hatchet: Hatchet,
) -> None:
    ref = await root_cancel_if_sleep.aio_run(wait_for_result=False)

    with pytest.raises(CancelledRunError):
        await ref.aio_result()

    details = await hatchet.runs.aio_get_details(ref.workflow_run_id)
    assert details.status == "CANCELLED"
