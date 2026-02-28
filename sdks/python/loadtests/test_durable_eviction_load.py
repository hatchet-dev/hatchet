"""
Load tests for durable eviction workflows.
"""

from __future__ import annotations

import asyncio
import time

import pytest

from hatchet_sdk import Hatchet
from hatchet_sdk.clients.admin import WorkflowRunDetail
from hatchet_sdk.clients.rest.api.task_api import TaskApi
from hatchet_sdk.clients.rest.models.v1_task_status import V1TaskStatus
from loadtests.worker import load_evictable_sleep, load_non_evictable_sleep

POLL_INTERVAL = 2
MAX_POLLS = 30


async def _poll_until_status(
    hatchet: Hatchet,
    workflow_run_id: str,
    target_status: V1TaskStatus,
) -> WorkflowRunDetail:
    for _ in range(MAX_POLLS):
        details = await hatchet.runs.aio_get_details(workflow_run_id)
        if any(t.status == target_status for t in details.task_runs.values()):
            return details
        await asyncio.sleep(POLL_INTERVAL)
    return await hatchet.runs.aio_get_details(workflow_run_id)


def _get_task_id(details: WorkflowRunDetail) -> str:
    return next(iter(details.task_runs.values())).external_id


@pytest.mark.loadtest
@pytest.mark.asyncio(loop_scope="session")
async def test_load_mass_eviction_restore(hatchet: Hatchet) -> None:
    """Evict 30 tasks, restore all, verify all complete."""
    n = 30
    refs = [load_evictable_sleep.run_no_wait() for _ in range(n)]

    for ref in refs:
        await _poll_until_status(hatchet, ref.workflow_run_id, V1TaskStatus.RUNNING)

    evicted: dict[str, str] = {}
    for ref in refs:
        details = await _poll_until_status(
            hatchet, ref.workflow_run_id, V1TaskStatus.EVICTED
        )
        task_id = _get_task_id(details)
        evicted[ref.workflow_run_id] = task_id

    with hatchet.runs.client() as client:
        task_api = TaskApi(client)
        for task_id in evicted.values():
            task_api.v1_task_restore(task=task_id)

    results = await asyncio.gather(*[ref.aio_result() for ref in refs])

    assert len(results) == n
    for r in results:
        assert r["status"] == "completed"


@pytest.mark.loadtest
@pytest.mark.asyncio(loop_scope="session")
async def test_load_mixed_evictable_non_evictable(hatchet: Hatchet) -> None:
    """Run 20 evictable and 20 non-evictable; non-evictable should never be evicted."""
    n = 20
    evictable_refs = [load_evictable_sleep.run_no_wait() for _ in range(n)]
    non_evictable_refs = [load_non_evictable_sleep.run_no_wait() for _ in range(n)]

    await asyncio.sleep(8)

    for ref in non_evictable_refs:
        details = await hatchet.runs.aio_get_details(ref.workflow_run_id)
        statuses = {t.status for t in details.task_runs.values()}
        assert (
            V1TaskStatus.EVICTED not in statuses
        ), f"Non-evictable task was evicted: {statuses}"

    evicted_details = {}
    for ref in evictable_refs:
        details = await _poll_until_status(
            hatchet, ref.workflow_run_id, V1TaskStatus.EVICTED
        )
        evicted_details[ref.workflow_run_id] = _get_task_id(details)

    with hatchet.runs.client() as client:
        task_api = TaskApi(client)
        for task_id in evicted_details.values():
            task_api.v1_task_restore(task=task_id)

    evictable_results = await asyncio.gather(
        *[ref.aio_result() for ref in evictable_refs]
    )
    non_evictable_results = await asyncio.gather(
        *[ref.aio_result() for ref in non_evictable_refs]
    )

    assert len(evictable_results) == n
    assert len(non_evictable_results) == n
    for r in evictable_results + non_evictable_results:
        assert r["status"] == "completed"


@pytest.mark.loadtest
@pytest.mark.asyncio(loop_scope="session")
async def test_load_eviction_restore_throughput(hatchet: Hatchet) -> None:
    """Measure restore-to-completion time for 50 evicted tasks."""
    n = 50
    refs = [load_evictable_sleep.run_no_wait() for _ in range(n)]

    for ref in refs:
        await _poll_until_status(hatchet, ref.workflow_run_id, V1TaskStatus.RUNNING)

    task_ids = []
    for ref in refs:
        details = await _poll_until_status(
            hatchet, ref.workflow_run_id, V1TaskStatus.EVICTED
        )
        task_ids.append(_get_task_id(details))

    restore_start = time.time()
    with hatchet.runs.client() as client:
        task_api = TaskApi(client)
        for task_id in task_ids:
            task_api.v1_task_restore(task=task_id)

    await asyncio.gather(*[ref.aio_result() for ref in refs])
    restore_elapsed = time.time() - restore_start

    assert (
        restore_elapsed < 120
    ), f"Restore+completion of {n} tasks took {restore_elapsed}s, expected < 120s"
