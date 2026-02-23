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
    evictable_sleep,
    non_evictable_sleep,
)
from hatchet_sdk import Hatchet
from hatchet_sdk.clients.admin import WorkflowRunDetail
from hatchet_sdk.clients.rest.api.task_api import TaskApi
from hatchet_sdk.clients.rest.models.v1_task_status import V1TaskStatus
from tests.worker_fixture import hatchet_worker

POLL_INTERVAL = 2
MAX_POLLS = 15


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


@pytest.mark.asyncio(loop_scope="session")
async def test_non_evictable_task_completes(hatchet: Hatchet) -> None:
    """A durable task with eviction disabled should finish normally."""
    result = await non_evictable_sleep.aio_run()

    assert result["status"] == "completed"
    runtime = result["runtime"]
    assert isinstance(runtime, (int, float))
    assert runtime >= 10


@pytest.mark.asyncio(loop_scope="session")
async def test_evictable_task_is_evicted(hatchet: Hatchet) -> None:
    """After the TTL, the eviction manager should evict the task.

    We verify by polling gRPC until the task status goes from RUNNING to EVICTED.
    """
    ref = evictable_sleep.run_no_wait()

    await _poll_until_status(hatchet, ref.workflow_run_id, V1TaskStatus.RUNNING)
    details = await _poll_until_status(
        hatchet, ref.workflow_run_id, V1TaskStatus.EVICTED
    )
    statuses = {t.status for t in details.task_runs.values()}

    assert (
        V1TaskStatus.EVICTED in statuses
    ), f"Expected EVICTED after eviction, got: {statuses}"


@pytest.mark.asyncio(loop_scope="session")
async def test_evictable_task_restore(hatchet: Hatchet) -> None:
    """After eviction, a REST restore should re-enqueue the task."""
    ref = evictable_sleep.run_no_wait()

    await _poll_until_status(hatchet, ref.workflow_run_id, V1TaskStatus.RUNNING)
    details = await _poll_until_status(
        hatchet, ref.workflow_run_id, V1TaskStatus.EVICTED
    )
    task_id = list(details.task_runs.values())[0].external_id

    with hatchet.runs.client() as client:
        TaskApi(client).v1_task_restore(task=task_id)

    details = await _poll_until_status(
        hatchet, ref.workflow_run_id, V1TaskStatus.RUNNING
    )
    statuses = {t.status for t in details.task_runs.values()}

    assert (
        V1TaskStatus.RUNNING in statuses
    ), f"Expected RUNNING after restore, got: {statuses}"


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
