import asyncio

import pytest

from examples.durable.worker import (EVENT_KEY, SLEEP_TIME, durable_workflow,
                                     wait_for_sleep_twice)
from hatchet_sdk import Hatchet


@pytest.mark.asyncio(loop_scope="session")
async def test_durable(hatchet: Hatchet) -> None:
    ref = durable_workflow.run_no_wait()

    await asyncio.sleep(SLEEP_TIME + 10)

    hatchet.event.push(EVENT_KEY, {"test": "test"})

    result = await ref.aio_result()

    workers = await hatchet.workers.aio_list()

    assert workers.rows

    active_workers = [w for w in workers.rows if w.status == "ACTIVE"]

    assert len(active_workers) >= 2
    assert any(
        w.name == hatchet.config.apply_namespace("e2e-test-worker")
        for w in active_workers
    )
    assert any(
        w.name == hatchet.config.apply_namespace("e2e-test-worker_durable")
        for w in active_workers
    )

    assert result["durable_task"]["status"] == "success"

    wait_group_1 = result["wait_for_or_group_1"]
    wait_group_2 = result["wait_for_or_group_2"]

    assert abs(wait_group_1["runtime"] - SLEEP_TIME) < 3

    assert wait_group_1["key"] == wait_group_2["key"]
    assert wait_group_1["key"] == "CREATE"
    assert "sleep" in wait_group_1["event_id"]
    assert "event" in wait_group_2["event_id"]

    wait_for_multi_sleep = result["wait_for_multi_sleep"]

    assert wait_for_multi_sleep["runtime"] > 3 * SLEEP_TIME


@pytest.mark.asyncio(loop_scope="session")
async def test_durable_sleep_cancel_replay(hatchet: Hatchet) -> None:
    first_sleep = await wait_for_sleep_twice.aio_run_no_wait()

    await asyncio.sleep(SLEEP_TIME / 2)

    await hatchet.runs.aio_cancel(first_sleep.workflow_run_id)

    await first_sleep.aio_result()

    await hatchet.runs.aio_replay(
        first_sleep.workflow_run_id,
    )

    second_sleep_result = await first_sleep.aio_result()

    """We've already slept for a little bit by the time the task is cancelled"""
    assert second_sleep_result["runtime"] <= SLEEP_TIME
