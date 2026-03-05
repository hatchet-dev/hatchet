import asyncio
import time

import pytest
from uuid import uuid4

from examples.durable.worker import (
    EVENT_KEY,
    SLEEP_TIME,
    REPLAY_RESET_SLEEP_TIME,
    durable_sleep_event_spawn,
    durable_with_bulk_spawn,
    durable_with_spawn,
    durable_workflow,
    wait_for_sleep_twice,
    durable_spawn_dag,
    durable_non_determinism,
    durable_replay_reset,
    memo_task,
    MemoInput,
)
from hatchet_sdk import Hatchet

requires_durable_eviction = pytest.mark.usefixtures("_skip_unless_durable_eviction")


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

    assert wait_group_1["key"] == wait_group_2["key"]
    assert wait_group_1["key"] == "CREATE"
    assert "sleep" in wait_group_1["event_id"]
    assert "event" in wait_group_2["event_id"]


@requires_durable_eviction
@pytest.mark.asyncio(loop_scope="session")
async def test_durable_sleep_cancel_replay(hatchet: Hatchet) -> None:
    first_sleep = await wait_for_sleep_twice.aio_run_no_wait()

    await asyncio.sleep(SLEEP_TIME / 2)

    await hatchet.runs.aio_cancel(first_sleep.workflow_run_id)

    await first_sleep.aio_result()

    replay_start = time.time()
    await hatchet.runs.aio_replay(
        first_sleep.workflow_run_id,
    )

    second_sleep_result = await first_sleep.aio_result()
    replay_elapsed = time.time() - replay_start

    assert second_sleep_result["runtime"] < SLEEP_TIME
    assert replay_elapsed <= SLEEP_TIME


@pytest.mark.asyncio(loop_scope="session")
async def test_durable_child_spawn() -> None:
    result = await durable_with_spawn.aio_run()

    assert result["child_output"] == {"message": "hello from child"}


@pytest.mark.asyncio(loop_scope="session")
async def test_durable_child_bulk_spawn() -> None:
    result = await durable_with_bulk_spawn.aio_run()

    assert result["child_outputs"] == [
        {"message": "hello from child"} for _ in range(10)
    ]


@requires_durable_eviction
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

    replay_start = time.time()
    await hatchet.runs.aio_replay(ref.workflow_run_id)
    replayed_result = await ref.aio_result()
    replay_elapsed = time.time() - replay_start

    assert replayed_result["child_output"] == {"message": "hello from child"}
    assert replay_elapsed < SLEEP_TIME


@requires_durable_eviction
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

    assert replayed_result["runtime"] < SLEEP_TIME
    assert elapsed < SLEEP_TIME


@pytest.mark.asyncio(loop_scope="session")
async def test_durable_spawn_dag() -> None:
    start = time.time()
    result = await durable_spawn_dag.aio_run()
    elapsed = time.time() - start

    assert result["sleep_duration"] >= 1
    assert result["spawn_duration"] >= 5
    assert elapsed >= 5
    assert elapsed <= 15


@requires_durable_eviction
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


@requires_durable_eviction
@pytest.mark.parametrize("node_id", [1, 2, 3])
@pytest.mark.asyncio(loop_scope="session")
async def test_durable_replay_reset(hatchet: Hatchet, node_id: int) -> None:
    ref = await durable_replay_reset.aio_run_no_wait()

    result = await ref.aio_result()

    assert result.sleep_1_duration >= REPLAY_RESET_SLEEP_TIME
    assert result.sleep_2_duration >= REPLAY_RESET_SLEEP_TIME
    assert result.sleep_3_duration >= REPLAY_RESET_SLEEP_TIME

    await hatchet.runs.aio_reset_durable_task(
        ref.workflow_run_id, node_id=node_id, branch_id=1
    )

    start = time.time()
    reset_result = await ref.aio_result()
    reset_elapsed = time.time() - start

    durations = [
        reset_result.sleep_1_duration,
        reset_result.sleep_2_duration,
        reset_result.sleep_3_duration,
    ]

    for i, duration in enumerate(durations, start=1):
        if i < node_id:
            assert duration < 1
        else:
            assert duration >= REPLAY_RESET_SLEEP_TIME

    sleeps_to_redo = 3 - node_id + 1
    assert reset_elapsed >= sleeps_to_redo * REPLAY_RESET_SLEEP_TIME


@requires_durable_eviction
@pytest.mark.asyncio(loop_scope="session")
async def test_durable_branching_off_branch(hatchet: Hatchet) -> None:
    ref = await durable_replay_reset.aio_run_no_wait()

    result = await ref.aio_result()

    assert result.sleep_1_duration >= REPLAY_RESET_SLEEP_TIME
    assert result.sleep_2_duration >= REPLAY_RESET_SLEEP_TIME
    assert result.sleep_3_duration >= REPLAY_RESET_SLEEP_TIME

    reset_from_node_id = 1

    await hatchet.runs.aio_reset_durable_task(
        ref.workflow_run_id, node_id=reset_from_node_id, branch_id=1
    )

    start = time.time()
    await asyncio.sleep(1)
    reset_result = await ref.aio_result()
    reset_elapsed = time.time() - start

    assert reset_result.sleep_1_duration >= REPLAY_RESET_SLEEP_TIME
    assert reset_result.sleep_2_duration >= REPLAY_RESET_SLEEP_TIME
    assert reset_result.sleep_3_duration >= REPLAY_RESET_SLEEP_TIME

    sleeps_to_redo = 3 - reset_from_node_id + 1
    assert reset_elapsed >= sleeps_to_redo * REPLAY_RESET_SLEEP_TIME

    reset_from_node_id = 2
    await hatchet.runs.aio_reset_durable_task(
        ## branch off branch 2 this time
        ref.workflow_run_id,
        node_id=reset_from_node_id,
        branch_id=2,
    )

    start = time.time()
    await asyncio.sleep(1)
    reset_result = await ref.aio_result()
    reset_elapsed = time.time() - start

    assert reset_result.sleep_1_duration < 1
    assert reset_result.sleep_2_duration >= REPLAY_RESET_SLEEP_TIME
    assert reset_result.sleep_3_duration >= REPLAY_RESET_SLEEP_TIME

    sleeps_to_redo = 3 - reset_from_node_id + 1
    assert reset_elapsed >= sleeps_to_redo * REPLAY_RESET_SLEEP_TIME


@requires_durable_eviction
@pytest.mark.asyncio(loop_scope="session")
async def test_durable_memoization_via_replay(hatchet: Hatchet) -> None:
    message = str(uuid4())
    start = time.time()
    ref = await memo_task.aio_run_no_wait(MemoInput(message=message))
    result_1 = await ref.aio_result()
    duration_1 = time.time() - start

    await hatchet.runs.aio_replay(ref.workflow_run_id)

    start = time.time()
    result_2 = await ref.aio_result()
    duration_2 = time.time() - start

    assert duration_1 >= SLEEP_TIME
    assert duration_2 < 1
    assert result_1.message == result_2.message
