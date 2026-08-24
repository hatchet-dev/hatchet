import asyncio
import pytest

from examples.bug_tests.test_durable_event_wait_scopes.worker import (
    EVENT_KEY,
    WaiterInput,
    scope_waiter,
)
from examples.test_utils import wait_for_running_status
from hatchet_sdk import Hatchet, TaskRunRef


async def push_event(hatchet: Hatchet, scope: str) -> None:
    await hatchet.event.aio_push(EVENT_KEY, {}, scope=scope)


async def run_tasks(
    hatchet: Hatchet,
    lookback_seconds: tuple[int, int] = (1, 1),
    scopes: tuple[str, str] = ("scope-a", "scope-b"),
) -> tuple[TaskRunRef[WaiterInput, None], TaskRunRef[WaiterInput, None]]:
    lba, lbb = lookback_seconds
    scope_a, scope_b = scopes

    ref_a = await scope_waiter.aio_run(
        WaiterInput(scope=scope_a, lookback_seconds=lba), wait_for_result=False
    )
    ref_b = await scope_waiter.aio_run(
        WaiterInput(scope=scope_b, lookback_seconds=lbb), wait_for_result=False
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

    await push_event(hatchet, scope="scope-a")

    await assert_on_results(refs)


@pytest.mark.asyncio(loop_scope="session")
async def test_durable_event_only_satisfied_on_matching_scope_lookback_path(
    hatchet: Hatchet,
) -> None:
    await push_event(hatchet, scope="scope-a")

    await asyncio.sleep(3)

    refs = await run_tasks(hatchet, lookback_seconds=(15, 15))

    await assert_on_results(refs)


@pytest.mark.asyncio(loop_scope="session")
async def test_durable_event_lookback_does_not_leak_across_same_scope_waiters(
    hatchet: Hatchet,
) -> None:
    shared_scope = "scope-shared-key-and-scope"

    await push_event(hatchet, scope=shared_scope)

    await asyncio.sleep(3)

    refs = await run_tasks(
        hatchet,
        lookback_seconds=(5, 1),
        scopes=(shared_scope, shared_scope),
    )

    await assert_on_results(refs)
