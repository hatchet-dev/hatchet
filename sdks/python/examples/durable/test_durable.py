import asyncio
import time

import pytest

from examples.durable.worker import (
    EVENT_KEY,
    SLEEP_TIME,
    REPLAY_RESET_SLEEP_TIME,
    durable_sleep_event_spawn,
    durable_with_spawn,
    durable_workflow,
    wait_for_sleep_twice,
    durable_spawn_dag,
    durable_non_determinism,
    durable_replay_reset,
)
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

    assert any(
        w.name == hatchet.config.apply_namespace("e2e-test-worker")
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


@pytest.mark.asyncio(loop_scope="session")
async def test_durable_child_spawn() -> None:
    result = await durable_with_spawn.aio_run()

    assert result["child_output"] == {"message": "hello from child"}


@pytest.mark.asyncio(loop_scope="session")
async def test_durable_sleep_event_spawn_replay(hatchet: Hatchet) -> None:
    start = time.time()
    ref = durable_sleep_event_spawn.run_no_wait()

    await asyncio.sleep(SLEEP_TIME + 5)
    hatchet.event.push(EVENT_KEY, {"test": "test"})

    result = await ref.aio_result()
    first_elapsed = time.time() - start

    assert result["child_output"] == {"message": "hello from child"}
    assert first_elapsed >= SLEEP_TIME
    assert result["runtime"] >= SLEEP_TIME

    replay_start = time.time()
    await hatchet.runs.aio_replay(ref.workflow_run_id)
    replayed_result = await ref.aio_result()
    replay_elapsed = time.time() - replay_start

    assert replayed_result["child_output"] == {"message": "hello from child"}
    assert replay_elapsed < SLEEP_TIME


@pytest.mark.asyncio(loop_scope="session")
async def test_durable_completed_replay(hatchet: Hatchet) -> None:
    ref = wait_for_sleep_twice.run_no_wait()

    start = time.time()
    first_result = await ref.aio_result()
    elapsed = time.time() - start

    assert first_result["runtime"] >= SLEEP_TIME
    assert elapsed >= SLEEP_TIME

    start = time.time()
    await hatchet.runs.aio_replay(ref.workflow_run_id)
    replayed_result = await ref.aio_result()
    elapsed = time.time() - start

    assert replayed_result["runtime"] >= 0
    assert replayed_result["runtime"] <= SLEEP_TIME

    assert elapsed < SLEEP_TIME


@pytest.mark.asyncio(loop_scope="session")
async def test_durable_spawn_dag() -> None:
    result = await durable_spawn_dag.aio_run()

    assert result["sleep_duration"] >= 1
    assert result["sleep_duration"] <= 2
    assert result["spawn_duration"] >= 5
    assert result["spawn_duration"] <= 10


@pytest.mark.asyncio(loop_scope="session")
async def test_durable_non_determinism(hatchet: Hatchet) -> None:
    ref = await durable_non_determinism.aio_run_no_wait()
    result = await ref.aio_result()

    assert result.sleep_time > result.attempt_number
    assert (  ## headroom to prevent flakiness
        result.sleep_time < result.attempt_number * 3
    )
    assert not result.non_determinism_detected

    await hatchet.runs.aio_replay(ref.workflow_run_id)

    replayed_result = await ref.aio_result()

    assert replayed_result.non_determinism_detected
    assert replayed_result.node_id == 1
    assert replayed_result.attempt_number == 2


@pytest.mark.parametrize("node_id", [1, 2, 3])
@pytest.mark.asyncio(loop_scope="session")
async def test_durable_replay_reset(hatchet: Hatchet, node_id: int) -> None:
    ref = await durable_replay_reset.aio_run_no_wait()
    result = await ref.aio_result()

    assert result.sleep_1_duration >= REPLAY_RESET_SLEEP_TIME
    assert result.sleep_2_duration >= REPLAY_RESET_SLEEP_TIME
    assert result.sleep_3_duration >= REPLAY_RESET_SLEEP_TIME

    await hatchet.runs.aio_reset_durable_task(ref.workflow_run_id, node_id=node_id)

    result = await ref.aio_result()

    for ix, dur in enumerate(
        [
            result.sleep_1_duration,
            result.sleep_2_duration,
            result.sleep_3_duration,
        ]
    ):
        if ix + 1 < node_id:
            assert dur < (REPLAY_RESET_SLEEP_TIME / 2)
        else:
            assert dur >= REPLAY_RESET_SLEEP_TIME
