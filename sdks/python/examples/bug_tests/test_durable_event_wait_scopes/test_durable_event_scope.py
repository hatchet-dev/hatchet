import asyncio
import pytest

from examples.bug_tests.test_durable_event_wait_scopes.worker import (
    EVENT_KEY,
    WaiterInput,
    scope_waiter,
)
from examples.test_utils import wait_for_running_status
from hatchet_sdk import Hatchet, TaskRunRef


async def run_tasks(
    hatchet: Hatchet,
) -> tuple[TaskRunRef[WaiterInput, None], TaskRunRef[WaiterInput, None]]:
    ref_a = await scope_waiter.aio_run(
        WaiterInput(scope="scope-a"), wait_for_result=False
    )
    ref_b = await scope_waiter.aio_run(
        WaiterInput(scope="scope-b"), wait_for_result=False
    )

    await wait_for_running_status(hatchet, ref_a.workflow_run_id, timeout=10)
    await wait_for_running_status(hatchet, ref_b.workflow_run_id, timeout=10)

    return ref_a, ref_b


async def assert_on_results(
    refs: tuple[TaskRunRef[WaiterInput, None], TaskRunRef[WaiterInput, None]],
) -> None:
    ref_a, ref_b = refs

    await ref_a.aio_result()

    with pytest.raises(asyncio.TimeoutError):
        await asyncio.wait_for(ref_b.aio_result(), timeout=5)


@pytest.mark.asyncio(loop_scope="session")
async def test_durable_event_only_satisfied_on_matching_scope_live_path(
    hatchet: Hatchet,
) -> None:
    refs = await run_tasks(hatchet)

    await hatchet.event.aio_push(
        EVENT_KEY, {"secret": "live-payload-for-scope-a"}, scope="scope-a"
    )

    await assert_on_results(refs)


@pytest.mark.asyncio(loop_scope="session")
async def test_durable_event_only_satisfied_on_matching_scope_lookback_path(
    hatchet: Hatchet,
) -> None:
    await hatchet.event.aio_push(
        EVENT_KEY, {"secret": "historical-payload-for-scope-a"}, scope="scope-a"
    )

    await asyncio.sleep(1)

    refs = await run_tasks(hatchet)

    await assert_on_results(refs)
