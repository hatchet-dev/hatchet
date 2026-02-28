"""
Load tests for durable workflows.
"""

from __future__ import annotations

import asyncio
import time
from typing import Any

import pytest

from hatchet_sdk import Hatchet
from loadtests.config import LoadTestConfig
from loadtests.worker import (
    load_durable_child_spawn,
    load_durable_sleep,
    load_durable_sleep_event,
)

LOAD_EVENT_KEY = "loadtest:event"


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
async def test_load_durable_sleep_concurrent(
    hatchet: Hatchet, load_config: LoadTestConfig
) -> None:
    """Enqueue N durable sleep tasks in parallel, then wait for all to complete."""
    n = load_config.n_durable_sleep
    refs = await asyncio.gather(
        *[load_durable_sleep.aio_run_no_wait() for _ in range(n)]
    )

    start = time.time()
    results = await _gather_results(refs, load_config.result_concurrency)
    elapsed = time.time() - start

    assert len(results) == n
    completed = sum(1 for r in results if r.get("status") == "completed")
    assert completed == n, f"{completed}/{n} completed"

    avg_ms = (elapsed / n) * 1000
    assert avg_ms < 5000, f"Average duration {avg_ms:.0f}ms exceeds 5s threshold"


@pytest.mark.loadtest
@pytest.mark.asyncio(loop_scope="session")
async def test_load_durable_sleep_event_concurrent(
    hatchet: Hatchet, load_config: LoadTestConfig
) -> None:
    """Enqueue N durable sleep+event tasks in parallel, publish events after all start."""
    n = load_config.n_durable_event
    refs = await asyncio.gather(
        *[load_durable_sleep_event.aio_run_no_wait() for _ in range(n)]
    )

    await asyncio.sleep(5)

    for _ in range(n):
        hatchet.event.push(LOAD_EVENT_KEY, {})

    results = await _gather_results(refs, load_config.result_concurrency)

    assert len(results) == n
    completed = sum(1 for r in results if r.get("status") == "completed")
    assert completed == n, f"{completed}/{n} completed"


@pytest.mark.loadtest
@pytest.mark.asyncio(loop_scope="session")
async def test_load_durable_child_spawn_concurrent(
    load_config: LoadTestConfig,
) -> None:
    """Enqueue N durable child spawn tasks in parallel, then wait for all to complete."""
    n = load_config.n_durable_child
    refs = await asyncio.gather(
        *[load_durable_child_spawn.aio_run_no_wait() for _ in range(n)]
    )

    start = time.time()
    results = await _gather_results(refs, load_config.result_concurrency)
    elapsed = time.time() - start

    assert len(results) == n
    for r in results:
        assert r["status"] == "completed"
        assert r["child"] == {"status": "done"}

    max_seconds = max(120, n * 2)
    assert (
        elapsed < max_seconds
    ), f"All {n} runs should complete within {max_seconds}s, took {elapsed:.0f}s"
