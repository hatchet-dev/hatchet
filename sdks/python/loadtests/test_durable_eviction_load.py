"""
Load tests for durable eviction workflows.

Tasks are evicted (TTL < sleep duration), then auto-restored by the server
when the sleep completes. No manual restore calls.
"""

from __future__ import annotations

import asyncio
import time
from typing import Any

import pytest

from hatchet_sdk import Hatchet
from hatchet_sdk.clients.admin import WorkflowRunDetail
from hatchet_sdk.clients.rest.models.v1_task_status import V1TaskStatus
from loadtests.config import LoadTestConfig
from loadtests.worker import load_evictable_sleep, load_non_evictable_sleep

POLL_INTERVAL = 2
MAX_POLLS = 60


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


async def _poll_many(
    hatchet: Hatchet,
    refs: list[Any],
    target_status: V1TaskStatus,
    concurrency: int,
) -> list[WorkflowRunDetail]:
    """Poll all refs in parallel with bounded concurrency."""
    sem = asyncio.Semaphore(concurrency)
    results: list[WorkflowRunDetail | None] = [None] * len(refs)

    async def _poll(idx: int) -> None:
        async with sem:
            results[idx] = await _poll_until_status(
                hatchet, refs[idx].workflow_run_id, target_status
            )

    await asyncio.gather(*[_poll(i) for i in range(len(refs))])
    return results  # type: ignore[return-value]


async def _gather_results(
    refs: list[Any],
    concurrency: int,
) -> list[dict[str, Any]]:
    """Gather aio_result() from refs with bounded concurrency."""
    sem = asyncio.Semaphore(concurrency)
    results: list[dict[str, Any]] = [{}] * len(refs)

    async def _get(idx: int) -> None:
        async with sem:
            results[idx] = await refs[idx].aio_result()

    await asyncio.gather(*[_get(i) for i in range(len(refs))])
    return results


@pytest.mark.loadtest
@pytest.mark.asyncio(loop_scope="session")
async def test_load_mass_eviction(
    hatchet: Hatchet, load_config: LoadTestConfig
) -> None:
    """Enqueue N evictable tasks in parallel, evict, auto-restore, verify all complete."""
    n = load_config.n_eviction
    refs = await asyncio.gather(
        *[load_evictable_sleep.aio_run_no_wait() for _ in range(n)]
    )

    evicted_details = await _poll_many(
        hatchet, refs, V1TaskStatus.EVICTED, load_config.poll_concurrency
    )
    evicted_count = sum(
        1
        for d in evicted_details
        if any(t.status == V1TaskStatus.EVICTED for t in d.task_runs.values())
    )
    assert evicted_count == n, f"Only {evicted_count}/{n} were evicted"

    results = await _gather_results(refs, load_config.result_concurrency)

    assert len(results) == n
    completed = sum(1 for r in results if r.get("status") == "completed")
    assert completed == n, f"{completed}/{n} completed"


@pytest.mark.loadtest
@pytest.mark.asyncio(loop_scope="session")
async def test_load_mixed_evictable_non_evictable(
    hatchet: Hatchet, load_config: LoadTestConfig
) -> None:
    """Enqueue N evictable + N non-evictable in parallel; non-evictable never evicted."""
    n = load_config.n_mixed
    evictable_refs, non_evictable_refs = await asyncio.gather(
        asyncio.gather(*[load_evictable_sleep.aio_run_no_wait() for _ in range(n)]),
        asyncio.gather(*[load_non_evictable_sleep.aio_run_no_wait() for _ in range(n)]),
    )

    await asyncio.sleep(8)

    sem = asyncio.Semaphore(load_config.poll_concurrency)

    async def _check_not_evicted(ref: Any) -> None:
        async with sem:
            details = await hatchet.runs.aio_get_details(ref.workflow_run_id)
            statuses = {t.status for t in details.task_runs.values()}
            assert (
                V1TaskStatus.EVICTED not in statuses
            ), f"Non-evictable task was evicted: {statuses}"

    await asyncio.gather(*[_check_not_evicted(r) for r in non_evictable_refs])

    evictable_results = await _gather_results(
        evictable_refs, load_config.result_concurrency
    )
    non_evictable_results = await _gather_results(
        non_evictable_refs, load_config.result_concurrency
    )

    assert len(evictable_results) == n
    assert len(non_evictable_results) == n
    all_results = evictable_results + non_evictable_results
    completed = sum(1 for r in all_results if r.get("status") == "completed")
    assert completed == n * 2, f"{completed}/{n * 2} completed"


@pytest.mark.loadtest
@pytest.mark.asyncio(loop_scope="session")
async def test_load_eviction_completion_throughput(
    hatchet: Hatchet, load_config: LoadTestConfig
) -> None:
    """Enqueue N evictable tasks in parallel, measure eviction-to-completion time."""
    n = load_config.n_throughput
    refs = await asyncio.gather(
        *[load_evictable_sleep.aio_run_no_wait() for _ in range(n)]
    )

    await _poll_many(hatchet, refs, V1TaskStatus.EVICTED, load_config.poll_concurrency)

    eviction_time = time.time()
    results = await _gather_results(refs, load_config.result_concurrency)
    completion_elapsed = time.time() - eviction_time

    assert len(results) == n
    completed = sum(1 for r in results if r.get("status") == "completed")
    assert completed == n, f"{completed}/{n} completed"

    max_seconds = max(120, n)
    assert completion_elapsed < max_seconds, (
        f"Eviction-to-completion of {n} tasks took {completion_elapsed:.0f}s, "
        f"expected < {max_seconds}s"
    )
