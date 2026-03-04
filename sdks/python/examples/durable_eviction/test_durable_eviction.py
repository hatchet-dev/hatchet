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

import psutil
import pytest

from examples.durable_eviction.worker import (
    EVENT_KEY,
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
from tests.worker_fixture import hatchet_worker

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
    ref = non_evictable_sleep.run_no_wait()

    await _poll_until_status(hatchet, ref.workflow_run_id, V1TaskStatus.RUNNING)
    await asyncio.sleep(7)  # Past EVICTION_TTL (5s), task still sleeping (10s total)
    details = await hatchet.runs.aio_get_details(ref.workflow_run_id)
    statuses = {t.status for t in details.task_runs.values()}
    assert (
        V1TaskStatus.EVICTED not in statuses
    ), f"Non-evictable task should never be EVICTED, got: {statuses}"

    result = await ref.aio_result()
    assert result["status"] == "completed"


@requires_durable_eviction
@pytest.mark.asyncio(loop_scope="session")
async def test_evictable_task_is_evicted(hatchet: Hatchet) -> None:
    """After the TTL, the eviction manager should evict the task."""
    ref = evictable_sleep.run_no_wait()

    await _poll_until_status(hatchet, ref.workflow_run_id, V1TaskStatus.RUNNING)
    details = await _poll_until_status(
        hatchet, ref.workflow_run_id, V1TaskStatus.EVICTED
    )
    statuses = {t.status for t in details.task_runs.values()}

    assert (
        V1TaskStatus.EVICTED in statuses
    ), f"Expected EVICTED after eviction, got: {statuses}"


@requires_durable_eviction
@pytest.mark.asyncio(loop_scope="session")
async def test_evictable_task_restore(hatchet: Hatchet) -> None:
    """After eviction, a REST restore should re-enqueue the task."""
    ref = evictable_sleep.run_no_wait()

    await _poll_until_status(hatchet, ref.workflow_run_id, V1TaskStatus.RUNNING)
    details = await _poll_until_status(
        hatchet, ref.workflow_run_id, V1TaskStatus.EVICTED
    )
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
    ref = evictable_sleep.run_no_wait()

    await _poll_until_status(hatchet, ref.workflow_run_id, V1TaskStatus.RUNNING)
    details = await _poll_until_status(
        hatchet, ref.workflow_run_id, V1TaskStatus.EVICTED
    )
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
    ref = evictable_wait_for_event.run_no_wait()

    await _poll_until_status(hatchet, ref.workflow_run_id, V1TaskStatus.RUNNING)
    details = await _poll_until_status(
        hatchet, ref.workflow_run_id, V1TaskStatus.EVICTED
    )
    statuses = {t.status for t in details.task_runs.values()}

    assert (
        V1TaskStatus.EVICTED in statuses
    ), f"Expected EVICTED for wait_for_event, got: {statuses}"


@requires_durable_eviction
@pytest.mark.asyncio(loop_scope="session")
async def test_evictable_wait_for_event_restore(hatchet: Hatchet) -> None:
    """After eviction, restoring and sending the event should let the task complete."""
    ref = evictable_wait_for_event.run_no_wait()

    await _poll_until_status(hatchet, ref.workflow_run_id, V1TaskStatus.RUNNING)
    details = await _poll_until_status(
        hatchet, ref.workflow_run_id, V1TaskStatus.EVICTED
    )
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
    ref = evictable_child_spawn.run_no_wait()

    await _poll_until_status(hatchet, ref.workflow_run_id, V1TaskStatus.RUNNING)
    details = await _poll_until_status(
        hatchet, ref.workflow_run_id, V1TaskStatus.EVICTED
    )
    statuses = {t.status for t in details.task_runs.values()}

    assert (
        V1TaskStatus.EVICTED in statuses
    ), f"Expected EVICTED for child_spawn, got: {statuses}"


@requires_durable_eviction
@pytest.mark.asyncio(loop_scope="session")
async def test_evictable_child_spawn_restore(hatchet: Hatchet) -> None:
    """After eviction, restoring should let the child-spawning task resume."""
    ref = evictable_child_spawn.run_no_wait()

    await _poll_until_status(hatchet, ref.workflow_run_id, V1TaskStatus.RUNNING)
    details = await _poll_until_status(
        hatchet, ref.workflow_run_id, V1TaskStatus.EVICTED
    )
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
    ref = evictable_child_spawn.run_no_wait()

    await _poll_until_status(hatchet, ref.workflow_run_id, V1TaskStatus.RUNNING)
    details = await _poll_until_status(
        hatchet, ref.workflow_run_id, V1TaskStatus.EVICTED
    )
    task_id = _get_task_id(details)

    with hatchet.runs.client() as client:
        TaskApi(client).v1_task_restore(task=task_id)

    result = await ref.aio_result()
    assert result["status"] == "completed"
    assert result["child"] == {"child_status": "completed"}


@requires_durable_eviction
@pytest.mark.asyncio(loop_scope="session")
async def test_multiple_eviction_cycle(hatchet: Hatchet) -> None:
    """The task should survive two eviction+restore cycles."""
    start = time.time()
    ref = multiple_eviction.run_no_wait()

    # --- first eviction cycle ---
    await _poll_until_status(hatchet, ref.workflow_run_id, V1TaskStatus.RUNNING)
    details = await _poll_until_status(
        hatchet, ref.workflow_run_id, V1TaskStatus.EVICTED
    )
    statuses = {t.status for t in details.task_runs.values()}
    assert V1TaskStatus.EVICTED in statuses, f"First eviction failed: {statuses}"

    task_id = _get_task_id(details)
    with hatchet.runs.client() as client:
        TaskApi(client).v1_task_restore(task=task_id)

    # --- second eviction cycle ---
    await _poll_until_status(hatchet, ref.workflow_run_id, V1TaskStatus.RUNNING)
    details = await _poll_until_status(
        hatchet, ref.workflow_run_id, V1TaskStatus.EVICTED
    )
    statuses = {t.status for t in details.task_runs.values()}
    assert V1TaskStatus.EVICTED in statuses, f"Second eviction failed: {statuses}"

    task_id = _get_task_id(details)
    with hatchet.runs.client() as client:
        TaskApi(client).v1_task_restore(task=task_id)

    # --- should complete after the second restore ---
    result = await ref.aio_result()
    elapsed = time.time() - start
    assert result["status"] == "completed"
    assert elapsed >= 30


@requires_durable_eviction
@pytest.mark.asyncio(loop_scope="session")
async def test_graceful_termination_evicts_waiting_runs(hatchet: Hatchet) -> None:
    """When a worker receives SIGTERM, all waiting durable runs should be evicted."""
    command = ["poetry", "run", "python", "-m", "examples.durable_eviction.worker"]
    with hatchet_worker(command, healthcheck_port=8004) as proc:
        ref = evictable_sleep.run_no_wait()

        await _poll_until_status(hatchet, ref.workflow_run_id, V1TaskStatus.RUNNING)

        parent = psutil.Process(proc.pid)
        for child in parent.children(recursive=True):
            child.send_signal(signal.SIGTERM)
        parent.send_signal(signal.SIGTERM)

        details = await _poll_until_status(
            hatchet, ref.workflow_run_id, V1TaskStatus.EVICTED
        )
        statuses = {t.status for t in details.task_runs.values()}

        assert (
            V1TaskStatus.EVICTED in statuses
        ), f"Expected EVICTED after SIGTERM, got: {statuses}"


@pytest.mark.asyncio(loop_scope="session")
async def test_eviction_plus_replay(hatchet: Hatchet) -> None:
    """After eviction, replay (not restore) should re-queue the run from the beginning."""
    ref = evictable_sleep.run_no_wait()

    await _poll_until_status(hatchet, ref.workflow_run_id, V1TaskStatus.RUNNING)
    await _poll_until_status(hatchet, ref.workflow_run_id, V1TaskStatus.EVICTED)

    await hatchet.runs.aio_replay(ref.workflow_run_id)

    result = await ref.aio_result()
    assert result["status"] == "completed"


@pytest.mark.asyncio(loop_scope="session")
async def test_evictable_cancel_after_eviction(hatchet: Hatchet) -> None:
    """Cancelling an evicted run should transition it to CANCELLED."""
    ref = evictable_sleep.run_no_wait()

    await _poll_until_status(hatchet, ref.workflow_run_id, V1TaskStatus.RUNNING)
    details = await _poll_until_status(
        hatchet, ref.workflow_run_id, V1TaskStatus.EVICTED
    )
    statuses = {t.status for t in details.task_runs.values()}
    assert V1TaskStatus.EVICTED in statuses, f"Expected EVICTED, got: {statuses}"

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


@pytest.mark.asyncio(loop_scope="session")
async def test_restore_idempotency(hatchet: Hatchet) -> None:
    """Restoring twice on the same evicted task should not cause duplicate execution."""
    ref = evictable_sleep.run_no_wait()

    await _poll_until_status(hatchet, ref.workflow_run_id, V1TaskStatus.RUNNING)
    details = await _poll_until_status(
        hatchet, ref.workflow_run_id, V1TaskStatus.EVICTED
    )
    task_id = _get_task_id(details)

    with hatchet.runs.client() as client:
        TaskApi(client).v1_task_restore(task=task_id)
        TaskApi(client).v1_task_restore(task=task_id)

    result = await ref.aio_result()
    assert result["status"] == "completed"
