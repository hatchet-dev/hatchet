"""
Load tests for durable workflows.
"""

from __future__ import annotations

import asyncio
import time

import pytest

from hatchet_sdk import Hatchet
from loadtests.worker import (
    load_durable_child_spawn,
    load_durable_sleep,
    load_durable_sleep_event,
)

LOAD_EVENT_KEY = "loadtest:event"


@pytest.mark.loadtest
@pytest.mark.asyncio(loop_scope="session")
async def test_load_durable_sleep_concurrent(hatchet: Hatchet) -> None:
    """Run 50 concurrent durable sleep tasks."""
    n = 50
    start = time.time()
    refs = [load_durable_sleep.run_no_wait() for _ in range(n)]
    results = await asyncio.gather(*[ref.aio_result() for ref in refs])
    elapsed = time.time() - start

    assert len(results) == n
    for r in results:
        assert r["status"] == "completed"

    avg_ms = (elapsed / n) * 1000
    assert avg_ms < 5000, f"Average duration {avg_ms}ms exceeds 5s threshold"


@pytest.mark.loadtest
@pytest.mark.asyncio(loop_scope="session")
async def test_load_durable_sleep_event_concurrent(hatchet: Hatchet) -> None:
    """Run 20 durable sleep+event tasks, publish events after all start."""
    n = 20
    refs = [load_durable_sleep_event.run_no_wait() for _ in range(n)]

    await asyncio.sleep(3)

    for _ in range(n):
        hatchet.event.push(LOAD_EVENT_KEY, {})

    results = await asyncio.gather(*[ref.aio_result() for ref in refs])

    assert len(results) == n
    for r in results:
        assert r["status"] == "completed"


@pytest.mark.loadtest
@pytest.mark.asyncio(loop_scope="session")
async def test_load_durable_child_spawn_concurrent() -> None:
    """Run 20 concurrent durable child spawn tasks."""
    n = 20
    start = time.time()
    refs = [load_durable_child_spawn.run_no_wait() for _ in range(n)]
    results = await asyncio.gather(*[ref.aio_result() for ref in refs])
    elapsed = time.time() - start

    assert len(results) == n
    for r in results:
        assert r["status"] == "completed"
        assert r["child"] == {"status": "done"}

    assert elapsed < 60, f"All {n} runs should complete within 60s, took {elapsed}s"
