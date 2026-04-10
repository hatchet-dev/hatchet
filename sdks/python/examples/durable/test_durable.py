import asyncio
import time

import pytest
from uuid import uuid4
import json
from typing import cast
from random import shuffle

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
    DurableBulkSpawnInput,
    memo_now_caching,
    AwaitedEvent,
    EventLookbackInput,
    wait_for_event_lookback,
    wait_for_or_event_lookback,
    wait_for_two_events_second_pushed_first,
)
from hatchet_sdk import Hatchet, PushEventOptions

requires_durable_eviction = pytest.mark.usefixtures("_skip_unless_durable_eviction")


@pytest.mark.asyncio(loop_scope="session")
async def test_durable_workflow(hatchet: Hatchet) -> None:
    ref = await durable_workflow.aio_run(wait_for_result=False)
    id = str(uuid4())

    await asyncio.sleep(SLEEP_TIME + 10)

    event = await hatchet.event.aio_push(
        EVENT_KEY, AwaitedEvent(id=id).model_dump(mode="json")
    )

    result = await ref.aio_result()

    workers = await hatchet.workers.aio_list()

    assert workers.rows

    active_workers = [w for w in workers.rows if w.status == "ACTIVE"]

    assert any(
        w.name == hatchet.config.apply_namespace("e2e-test-worker")
        for w in active_workers
    )

    assert result["durable_task"]["status"] == "success"

    # hack for old engine test
    assert (
        result["durable_task"]["event_id"] == ""
        or result["durable_task"]["event_id"] == id
    )
    assert result["durable_task"]["sleep_duration_seconds"] == SLEEP_TIME

    wait_group_1 = result["wait_for_or_group_1"]
    wait_group_2 = result["wait_for_or_group_2"]

    assert wait_group_1["key"] == wait_group_2["key"]
    assert wait_group_1["key"] == "CREATE"
    assert "sleep" in wait_group_1["event_id"]
    assert "event" in wait_group_2["event_id"]


@requires_durable_eviction
@pytest.mark.asyncio(loop_scope="session")
async def test_durable_sleep_cancel_replay(hatchet: Hatchet) -> None:
    first_sleep = await wait_for_sleep_twice.aio_run(wait_for_result=False)

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

    assert result["child_output"] == {"message": "hello from child 1"}


@pytest.mark.asyncio(loop_scope="session")
async def test_durable_child_bulk_spawn() -> None:
    n = 10
    result = await durable_with_bulk_spawn.aio_run(DurableBulkSpawnInput(n=n))

    assert result["child_outputs"] == [
        {"message": "hello from child " + str(i)} for i in range(n)
    ]


@requires_durable_eviction
@pytest.mark.asyncio(loop_scope="session")
async def test_durable_sleep_event_spawn_replay(hatchet: Hatchet) -> None:
    start = time.time()
    ref = await durable_sleep_event_spawn.aio_run(wait_for_result=False)

    await asyncio.sleep(SLEEP_TIME + 5)
    hatchet.event.push(EVENT_KEY, {"test": "test"})

    result = await ref.aio_result()
    first_elapsed = time.time() - start

    assert result["child_output"] == {"message": "hello from child 1"}
    assert first_elapsed >= SLEEP_TIME

    replay_start = time.time()
    await hatchet.runs.aio_replay(ref.workflow_run_id)
    replayed_result = await ref.aio_result()
    replay_elapsed = time.time() - replay_start

    assert replayed_result["child_output"] == {"message": "hello from child 1"}
    assert replay_elapsed < SLEEP_TIME


@requires_durable_eviction
@pytest.mark.asyncio(loop_scope="session")
async def test_durable_completed_replay(hatchet: Hatchet) -> None:
    ref = await wait_for_sleep_twice.aio_run(wait_for_result=False)

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
    ref = await durable_non_determinism.aio_run(wait_for_result=False)
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
    ref = await durable_replay_reset.aio_run(wait_for_result=False)

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
            assert duration < REPLAY_RESET_SLEEP_TIME
        else:
            assert duration >= REPLAY_RESET_SLEEP_TIME

    sleeps_to_redo = 3 - node_id + 1
    assert reset_elapsed >= sleeps_to_redo * REPLAY_RESET_SLEEP_TIME


@requires_durable_eviction
@pytest.mark.asyncio(loop_scope="session")
async def test_durable_branching_off_branch(hatchet: Hatchet) -> None:
    ref = await durable_replay_reset.aio_run(wait_for_result=False)

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

    assert reset_result.sleep_1_duration < REPLAY_RESET_SLEEP_TIME
    assert reset_result.sleep_2_duration >= REPLAY_RESET_SLEEP_TIME
    assert reset_result.sleep_3_duration >= REPLAY_RESET_SLEEP_TIME

    sleeps_to_redo = 3 - reset_from_node_id + 1
    assert reset_elapsed >= sleeps_to_redo * REPLAY_RESET_SLEEP_TIME


@requires_durable_eviction
@pytest.mark.asyncio(loop_scope="session")
async def test_durable_memoization_via_replay(hatchet: Hatchet) -> None:
    message = str(uuid4())
    start = time.time()
    ref = await memo_task.aio_run(MemoInput(message=message), wait_for_result=False)
    result_1 = await ref.aio_result()
    duration_1 = time.time() - start

    await hatchet.runs.aio_replay(ref.workflow_run_id)

    start = time.time()
    result_2 = await ref.aio_result()
    duration_2 = time.time() - start

    assert duration_1 >= SLEEP_TIME
    assert duration_2 < 1
    assert result_1.message == result_2.message


@requires_durable_eviction
@pytest.mark.asyncio(loop_scope="session")
async def test_durable_memo_now_caching(hatchet: Hatchet) -> None:
    ref = await memo_now_caching.aio_run(wait_for_result=False)

    result_1 = await ref.aio_result()

    await hatchet.runs.aio_replay(ref.workflow_run_id)

    result_2 = await ref.aio_result()

    assert result_1["start_time"] == result_2["start_time"]


@pytest.mark.asyncio(loop_scope="session")
async def test_event_lookback_before_wait(hatchet: Hatchet) -> None:
    scope = str(uuid4())

    hatchet.event.push(
        EVENT_KEY, {"order": "first"}, options=PushEventOptions(scope=scope)
    )

    await asyncio.sleep(1)

    result = await wait_for_event_lookback.aio_run(EventLookbackInput(scope=scope))

    assert (
        result.elapsed < 1
    ), "Event lookback should find the event that was pushed before the wait started, so should be basically instantaneous"
    assert result.event.order == "first"


@pytest.mark.asyncio(loop_scope="session")
async def test_or_group_event_lookback_before_wait(hatchet: Hatchet) -> None:
    scope = str(uuid4())

    hatchet.event.push(
        EVENT_KEY, {"order": "first"}, options=PushEventOptions(scope=scope)
    )
    await asyncio.sleep(1)

    result = await wait_for_or_event_lookback.aio_run(EventLookbackInput(scope=scope))

    assert result.elapsed < SLEEP_TIME


@pytest.mark.asyncio(loop_scope="session")
async def test_two_event_waits_second_pushed_first(hatchet: Hatchet) -> None:
    scope = str(uuid4())

    hatchet.event.push(
        "key2",
        {"order": "second"},
        options=PushEventOptions(scope=scope),
    )
    await asyncio.sleep(1)

    ref = await wait_for_two_events_second_pushed_first.aio_run_no_wait(
        EventLookbackInput(scope=scope)
    )

    await asyncio.sleep(3)

    hatchet.event.push(
        "key1",
        {"order": "first"},
        options=PushEventOptions(scope=scope),
    )

    result = await ref.aio_result()

    assert result.elapsed < SLEEP_TIME
    assert result.event1.order == "first"
    assert result.event2.order == "second"


@pytest.mark.asyncio(loop_scope="session")
async def test_engine_picks_most_recent_event(hatchet: Hatchet) -> None:
    scope = str(uuid4())
    key = str(uuid4())

    event = None
    iters = list(range(100))
    shuffle(iters)

    for i in iters:
        event = hatchet.event.push(
            key,
            {"order": str(i)},
            options=PushEventOptions(scope=scope),
        )

    assert event
    await asyncio.sleep(1)

    res = await wait_for_event_lookback.aio_run(
        EventLookbackInput(scope=scope, key=key)
    )

    payload = cast(dict[str, str], json.loads(event.payload))

    assert res.event.order == payload["order"]
