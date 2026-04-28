"""
Integration tests for durable slot eviction.

Run with:
    cd sdks/python
    poetry run pytest examples/durable_eviction/test_durable_eviction.py -v -s
"""

from __future__ import annotations

import asyncio
import signal
import time
from subprocess import Popen
from typing import Any

import psutil
import pytest

from examples.durable_eviction.worker import (
    EVENT_KEY,
    capacity_evictable_sleep,
    evictable_child_bulk_spawn,
    evictable_child_spawn,
    evictable_sleep,
    evictable_wait_for_event,
    multiple_eviction,
    non_evictable_sleep,
)
from hatchet_sdk import Hatchet
from hatchet_sdk.clients.admin import WorkflowRunDetail
from hatchet_sdk.clients.rest.api.task_api import TaskApi
from hatchet_sdk.clients.rest.models.v1_task_status import V1TaskStatus

POLL_INTERVAL = 0.2
MAX_POLLS = 150

requires_durable_eviction = pytest.mark.usefixtures("_skip_unless_durable_eviction")


async def _poll_until_status(
    hatchet: Hatchet,
    workflow_run_id: str,
    target_status: V1TaskStatus,
) -> WorkflowRunDetail:
    """Poll gRPC run details until any task reaches *target_status* (or timeout)."""
    for _ in range(MAX_POLLS):
        details = await hatchet.runs.aio_get_details(workflow_run_id)
        if any(t.status == target_status for t in details.task_runs.values()):
            return details
        await asyncio.sleep(POLL_INTERVAL)

    return await hatchet.runs.aio_get_details(workflow_run_id)


async def _poll_until_evicted(
    hatchet: Hatchet,
    workflow_run_id: str,
) -> WorkflowRunDetail:
    """Poll gRPC run details until any task has is_evicted=True (or timeout)."""
    for _ in range(MAX_POLLS):
        details = await hatchet.runs.aio_get_details(workflow_run_id)
        if any(t.is_evicted for t in details.task_runs.values()):
            return details
        await asyncio.sleep(POLL_INTERVAL)

    return await hatchet.runs.aio_get_details(workflow_run_id)


def _has_evicted_task(details: WorkflowRunDetail) -> bool:
    return any(t.is_evicted for t in details.task_runs.values())


def _get_task_id(details: WorkflowRunDetail) -> str:
    return list(details.task_runs.values())[0].external_id


@requires_durable_eviction
@pytest.mark.asyncio(loop_scope="session")
async def test_non_evictable_task_completes(hatchet: Hatchet) -> None:
    """A durable task with eviction disabled should finish normally."""
    start = time.time()
    result = await non_evictable_sleep.aio_run()
    elapsed = time.time() - start

    assert result["status"] == "completed"
    assert elapsed >= 10


@requires_durable_eviction
@pytest.mark.asyncio(loop_scope="session")
async def test_non_evictable_task_not_evicted(hatchet: Hatchet) -> None:
    """A durable task with eviction disabled should never be evicted, even past TTL."""
    ref = await non_evictable_sleep.aio_run(wait_for_result=False)

    await _poll_until_status(hatchet, ref.workflow_run_id, V1TaskStatus.RUNNING)
    await asyncio.sleep(7)  # Past EVICTION_TTL (5s), task still sleeping (10s total)
    details = await hatchet.runs.aio_get_details(ref.workflow_run_id)
    assert not _has_evicted_task(
        details
    ), f"Non-evictable task should never be evicted, got is_evicted=True"

    result = await ref.aio_result()
    assert result["status"] == "completed"


@requires_durable_eviction
@pytest.mark.asyncio(loop_scope="session")
async def test_evictable_task_is_evicted(hatchet: Hatchet) -> None:
    """After the TTL, the eviction manager should evict the task."""
    ref = await evictable_sleep.aio_run(wait_for_result=False)

    await _poll_until_status(hatchet, ref.workflow_run_id, V1TaskStatus.RUNNING)
    details = await _poll_until_evicted(hatchet, ref.workflow_run_id)

    assert _has_evicted_task(details), f"Expected is_evicted=True after eviction"


@requires_durable_eviction
@pytest.mark.asyncio(loop_scope="session")
async def test_evictable_task_restore(hatchet: Hatchet) -> None:
    """After eviction, a REST restore should re-enqueue the task."""
    ref = await evictable_sleep.aio_run(wait_for_result=False)

    await _poll_until_status(hatchet, ref.workflow_run_id, V1TaskStatus.RUNNING)
    details = await _poll_until_evicted(hatchet, ref.workflow_run_id)
    task_id = _get_task_id(details)

    with hatchet.runs.client() as client:
        TaskApi(client).v1_task_restore(task=task_id)

    details = await _poll_until_status(
        hatchet, ref.workflow_run_id, V1TaskStatus.RUNNING
    )
    statuses = {t.status for t in details.task_runs.values()}

    assert (
        V1TaskStatus.RUNNING in statuses
    ), f"Expected RUNNING after restore, got: {statuses}"


@requires_durable_eviction
@pytest.mark.asyncio(loop_scope="session")
async def test_evictable_task_restore_completes(hatchet: Hatchet) -> None:
    """After eviction and restore, evictable_sleep should complete and return a result."""
    start = time.time()
    ref = await evictable_sleep.aio_run(wait_for_result=False)

    await _poll_until_status(hatchet, ref.workflow_run_id, V1TaskStatus.RUNNING)
    details = await _poll_until_evicted(hatchet, ref.workflow_run_id)
    task_id = _get_task_id(details)

    with hatchet.runs.client() as client:
        TaskApi(client).v1_task_restore(task=task_id)

    result = await ref.aio_result()
    elapsed = time.time() - start
    assert result["status"] == "completed"
    assert elapsed >= 15


@requires_durable_eviction
@pytest.mark.asyncio(loop_scope="session")
async def test_evictable_wait_for_event_is_evicted(hatchet: Hatchet) -> None:
    """A durable task waiting for an event should be evicted after TTL."""
    ref = await evictable_wait_for_event.aio_run(wait_for_result=False)

    await _poll_until_status(hatchet, ref.workflow_run_id, V1TaskStatus.RUNNING)
    details = await _poll_until_evicted(hatchet, ref.workflow_run_id)

    assert _has_evicted_task(details), f"Expected is_evicted=True for wait_for_event"


@requires_durable_eviction
@pytest.mark.asyncio(loop_scope="session")
async def test_evictable_wait_for_event_restore(hatchet: Hatchet) -> None:
    """After eviction, restoring and sending the event should let the task complete."""
    ref = await evictable_wait_for_event.aio_run(wait_for_result=False)

    await _poll_until_status(hatchet, ref.workflow_run_id, V1TaskStatus.RUNNING)
    details = await _poll_until_evicted(hatchet, ref.workflow_run_id)
    task_id = _get_task_id(details)

    with hatchet.runs.client() as client:
        TaskApi(client).v1_task_restore(task=task_id)

    await _poll_until_status(hatchet, ref.workflow_run_id, V1TaskStatus.RUNNING)

    hatchet.event.push(EVENT_KEY, {})

    result = await ref.aio_result()
    assert result["status"] == "completed"


@requires_durable_eviction
@pytest.mark.asyncio(loop_scope="session")
async def test_evictable_child_spawn_is_evicted(hatchet: Hatchet) -> None:
    """A durable task waiting on a child workflow should be evicted after TTL."""
    ref = await evictable_child_spawn.aio_run(wait_for_result=False)

    await _poll_until_status(hatchet, ref.workflow_run_id, V1TaskStatus.RUNNING)
    details = await _poll_until_evicted(hatchet, ref.workflow_run_id)

    assert _has_evicted_task(details), f"Expected is_evicted=True for child_spawn"


@requires_durable_eviction
@pytest.mark.asyncio(loop_scope="session")
async def test_evictable_child_spawn_restore(hatchet: Hatchet) -> None:
    """After eviction, restoring should let the child-spawning task resume."""
    ref = await evictable_child_spawn.aio_run(wait_for_result=False)

    await _poll_until_status(hatchet, ref.workflow_run_id, V1TaskStatus.RUNNING)
    details = await _poll_until_evicted(hatchet, ref.workflow_run_id)
    task_id = _get_task_id(details)

    with hatchet.runs.client() as client:
        TaskApi(client).v1_task_restore(task=task_id)

    details = await _poll_until_status(
        hatchet, ref.workflow_run_id, V1TaskStatus.RUNNING
    )
    statuses = {t.status for t in details.task_runs.values()}

    assert (
        V1TaskStatus.RUNNING in statuses
    ), f"Expected RUNNING after restore, got: {statuses}"


@requires_durable_eviction
@pytest.mark.asyncio(loop_scope="session")
async def test_evictable_child_spawn_restore_completes(hatchet: Hatchet) -> None:
    """After eviction and restore, evictable_child_spawn should complete with child result."""
    ref = await evictable_child_spawn.aio_run(wait_for_result=False)

    await _poll_until_status(hatchet, ref.workflow_run_id, V1TaskStatus.RUNNING)
    details = await _poll_until_evicted(hatchet, ref.workflow_run_id)
    task_id = _get_task_id(details)

    with hatchet.runs.client() as client:
        TaskApi(client).v1_task_restore(task=task_id)

    result = await ref.aio_result()
    assert result["status"] == "completed"
    assert result["child"] == {"child_status": "completed"}


@requires_durable_eviction
@pytest.mark.asyncio(loop_scope="session")
async def test_evictable_child_bulk_spawn_restore_completes(hatchet: Hatchet) -> None:
    ref = await evictable_child_bulk_spawn.aio_run(wait_for_result=False)

    eviction_count = 0
    for _ in range(3):
        await _poll_until_status(hatchet, ref.workflow_run_id, V1TaskStatus.RUNNING)
        details = await _poll_until_evicted(hatchet, ref.workflow_run_id)
        eviction_count += 1
        task_id = _get_task_id(details)
        with hatchet.runs.client() as client:
            TaskApi(client).v1_task_restore(task=task_id)

    result = await ref.aio_result()
    assert eviction_count == 3, f"Expected 3 evictions, got {eviction_count}"
    assert result["child_results"] == [
        {"sleep_for": 10, "status": "completed"},
        {"sleep_for": 20, "status": "completed"},
        {"sleep_for": 30, "status": "completed"},
    ]


@requires_durable_eviction
@pytest.mark.asyncio(loop_scope="session")
async def test_multiple_eviction_cycle(hatchet: Hatchet) -> None:
    """The task should survive two eviction+restore cycles."""
    start = time.time()
    ref = await multiple_eviction.aio_run(wait_for_result=False)

    # --- first eviction cycle ---
    await _poll_until_status(hatchet, ref.workflow_run_id, V1TaskStatus.RUNNING)
    details = await _poll_until_evicted(hatchet, ref.workflow_run_id)
    assert _has_evicted_task(details), f"First eviction failed"

    task_id = _get_task_id(details)
    with hatchet.runs.client() as client:
        TaskApi(client).v1_task_restore(task=task_id)

    # --- second eviction cycle ---
    await _poll_until_status(hatchet, ref.workflow_run_id, V1TaskStatus.RUNNING)
    details = await _poll_until_evicted(hatchet, ref.workflow_run_id)
    assert _has_evicted_task(details), f"Second eviction failed"

    task_id = _get_task_id(details)
    with hatchet.runs.client() as client:
        TaskApi(client).v1_task_restore(task=task_id)

    # --- should complete after the second restore ---
    result = await ref.aio_result()
    elapsed = time.time() - start
    assert result["status"] == "completed"
    assert elapsed >= 30


@requires_durable_eviction
@pytest.mark.parametrize(
    "on_demand_worker",
    [
        ["poetry", "run", "python", "-m", "examples.durable_eviction.worker"],
    ],
    indirect=True,
)
@pytest.mark.asyncio(loop_scope="session")
async def test_graceful_termination_evicts_waiting_runs(
    hatchet: Hatchet, on_demand_worker: Popen[Any]
) -> None:
    """When a worker receives SIGTERM, all waiting durable runs should be evicted."""
    ref = await evictable_sleep.aio_run(wait_for_result=False)

    await _poll_until_status(hatchet, ref.workflow_run_id, V1TaskStatus.RUNNING)

    parent = psutil.Process(on_demand_worker.pid)
    for child in parent.children(recursive=True):
        child.send_signal(signal.SIGTERM)
    parent.send_signal(signal.SIGTERM)

    details = await _poll_until_evicted(hatchet, ref.workflow_run_id)

    assert _has_evicted_task(details), f"Expected is_evicted=True after SIGTERM"


@requires_durable_eviction
@pytest.mark.asyncio(loop_scope="session")
async def test_eviction_plus_replay(hatchet: Hatchet) -> None:
    """After eviction, replay (not restore) should re-queue the run from the beginning."""
    ref = await evictable_sleep.aio_run(wait_for_result=False)

    await _poll_until_status(hatchet, ref.workflow_run_id, V1TaskStatus.RUNNING)
    await _poll_until_evicted(hatchet, ref.workflow_run_id)

    await hatchet.runs.aio_replay(ref.workflow_run_id)

    result = await ref.aio_result()
    assert result["status"] == "completed"


@requires_durable_eviction
@pytest.mark.asyncio(loop_scope="session")
async def test_evictable_cancel_after_eviction(hatchet: Hatchet) -> None:
    """Cancelling an evicted run should transition it to CANCELLED."""
    ref = await evictable_sleep.aio_run(wait_for_result=False)

    await _poll_until_status(hatchet, ref.workflow_run_id, V1TaskStatus.RUNNING)
    details = await _poll_until_evicted(hatchet, ref.workflow_run_id)
    assert _has_evicted_task(details), f"Expected is_evicted=True"

    await hatchet.runs.aio_cancel(ref.workflow_run_id)

    status = await hatchet.runs.aio_get_status(ref.workflow_run_id)
    for _ in range(MAX_POLLS):
        status = await hatchet.runs.aio_get_status(ref.workflow_run_id)
        if status == V1TaskStatus.CANCELLED:
            break
        await asyncio.sleep(POLL_INTERVAL)
    else:
        status = await hatchet.runs.aio_get_status(ref.workflow_run_id)

    assert status == V1TaskStatus.CANCELLED


@requires_durable_eviction
@pytest.mark.parametrize(
    "on_demand_worker",
    [
        [
            "poetry",
            "run",
            "python",
            "-m",
            "examples.durable_eviction.capacity_worker",
        ]
    ],
    indirect=True,
)
@pytest.mark.asyncio(loop_scope="session")
async def test_capacity_eviction_fires(
    hatchet: Hatchet, on_demand_worker: Popen[Any]
) -> None:
    """A task with ttl=None but allow_capacity_eviction=True should be evicted
    under durable-slot pressure (durable_slots=1)."""
    ref = await capacity_evictable_sleep.aio_run(wait_for_result=False)

    await _poll_until_status(hatchet, ref.workflow_run_id, V1TaskStatus.RUNNING)
    details = await _poll_until_evicted(hatchet, ref.workflow_run_id)

    assert _has_evicted_task(
        details
    ), f"Expected capacity eviction (ttl=None), got no evicted tasks"


@requires_durable_eviction
@pytest.mark.parametrize(
    "on_demand_worker",
    [
        [
            "poetry",
            "run",
            "python",
            "-m",
            "examples.durable_eviction.capacity_worker",
        ]
    ],
    indirect=True,
)
@pytest.mark.asyncio(loop_scope="session")
async def test_capacity_eviction_restore_completes(
    hatchet: Hatchet, on_demand_worker: Popen[Any]
) -> None:
    """After capacity eviction, restore should let the task resume and complete."""
    ref = await capacity_evictable_sleep.aio_run(wait_for_result=False)

    await _poll_until_status(hatchet, ref.workflow_run_id, V1TaskStatus.RUNNING)
    details = await _poll_until_evicted(hatchet, ref.workflow_run_id)
    task_id = _get_task_id(details)

    with hatchet.runs.client() as client:
        TaskApi(client).v1_task_restore(task=task_id)

    result = await ref.aio_result()
    assert result["status"] == "completed"


@requires_durable_eviction
@pytest.mark.asyncio(loop_scope="session")
async def test_restore_idempotency(hatchet: Hatchet) -> None:
    """Restoring twice on the same evicted task should not cause duplicate execution."""
    ref = await evictable_sleep.aio_run(wait_for_result=False)

    await _poll_until_status(hatchet, ref.workflow_run_id, V1TaskStatus.RUNNING)
    details = await _poll_until_evicted(hatchet, ref.workflow_run_id)
    task_id = _get_task_id(details)

    with hatchet.runs.client() as client:
        TaskApi(client).v1_task_restore(task=task_id)
        TaskApi(client).v1_task_restore(task=task_id)

    result = await ref.aio_result()
    assert result["status"] == "completed"
