"""
Integration tests for durable slot eviction.

Run with:
    cd sdks/python
    poetry run pytest examples/durable_eviction/test_durable_eviction.py -v -s
"""

from __future__ import annotations

import asyncio
import signal

import psutil
import pytest

from examples.durable_eviction.worker import (
    EVICTION_TTL_SECONDS,
    evictable_sleep,
    non_evictable_sleep,
)
from hatchet_sdk import Hatchet
from hatchet_sdk.clients.rest.api.task_api import TaskApi
from hatchet_sdk.clients.rest.models.v1_task_event_type import V1TaskEventType
from tests.worker_fixture import hatchet_worker

POLL_INTERVAL = 2
MAX_POLLS = 15


async def _poll_for_event(
    hatchet: Hatchet,
    workflow_run_id: str,
    event_type: V1TaskEventType,
) -> set[V1TaskEventType]:
    """Poll run details until *event_type* appears (or timeout)."""
    for _ in range(MAX_POLLS):
        details = await hatchet.runs.aio_get(workflow_run_id)
        event_types = {e.event_type for e in details.task_events}
        if event_type in event_types:
            return event_types
        await asyncio.sleep(POLL_INTERVAL)

    details = await hatchet.runs.aio_get(workflow_run_id)
    return {e.event_type for e in details.task_events}


@pytest.mark.asyncio(loop_scope="session")
async def test_non_evictable_task_completes(hatchet: Hatchet) -> None:
    """A durable task with eviction disabled should finish normally."""
    result = await non_evictable_sleep.aio_run()

    assert result["status"] == "completed"
    assert result["runtime"] >= 10


@pytest.mark.asyncio(loop_scope="session")
async def test_evictable_task_is_evicted(hatchet: Hatchet) -> None:
    """After the TTL, the eviction manager should evict the task.

    We verify by polling the OLAP events until we see DURABLE_EVICTED.
    """
    ref = evictable_sleep.run_no_wait()

    event_types = await _poll_for_event(
        hatchet, ref.workflow_run_id, V1TaskEventType.DURABLE_EVICTED
    )

    assert V1TaskEventType.DURABLE_EVICTED in event_types, (
        f"Expected DURABLE_EVICTED in events, got: {event_types}"
    )


@pytest.mark.asyncio(loop_scope="session")
async def test_evictable_task_restore(hatchet: Hatchet) -> None:
    """After eviction, a REST restore should re-enqueue the task."""
    ref = evictable_sleep.run_no_wait()

    event_types = await _poll_for_event(
        hatchet, ref.workflow_run_id, V1TaskEventType.DURABLE_EVICTED
    )
    assert V1TaskEventType.DURABLE_EVICTED in event_types

    details = await hatchet.runs.aio_get(ref.workflow_run_id)
    task_id = details.tasks[0].metadata.id

    with hatchet.runs.client() as client:
        TaskApi(client).v1_task_restore(task=task_id)

    event_types = await _poll_for_event(
        hatchet, ref.workflow_run_id, V1TaskEventType.DURABLE_RESTORING
    )

    assert V1TaskEventType.DURABLE_RESTORING in event_types, (
        f"Expected DURABLE_RESTORING after restore, got: {event_types}"
    )


@pytest.mark.asyncio(loop_scope="session")
async def test_graceful_termination_evicts_waiting_runs(hatchet: Hatchet) -> None:
    """When a worker receives SIGTERM, all waiting durable runs should be evicted."""
    # Start a dedicated worker for this test (separate from the session-scoped one)
    # so we can SIGTERM it without affecting other tests.
    command = ["poetry", "run", "python", "-m", "examples.durable_eviction.worker"]
    with hatchet_worker(command, healthcheck_port=8004) as proc:
        ref = evictable_sleep.run_no_wait()

        # Wait long enough for the task to be picked up and enter the durable sleep
        await asyncio.sleep(5)

        # Send SIGTERM to trigger graceful shutdown
        parent = psutil.Process(proc.pid)
        for child in parent.children(recursive=True):
            child.send_signal(signal.SIGTERM)
        parent.send_signal(signal.SIGTERM)

        event_types = await _poll_for_event(
            hatchet, ref.workflow_run_id, V1TaskEventType.DURABLE_EVICTED
        )

        assert V1TaskEventType.DURABLE_EVICTED in event_types, (
            f"Expected DURABLE_EVICTED after SIGTERM, got: {event_types}"
        )
