from __future__ import annotations

from typing import Any

import pytest

from examples.durable_complex.execution_timeout.worker import (
    durable_refresh_timeout_workflow,
    durable_timeout_completes_workflow,
    durable_timeout_eviction_workflow,
    durable_timeout_workflow,
)


@pytest.mark.asyncio(loop_scope="session")
async def test_durable_execution_timeout_exceeded() -> None:
    """Durable task that sleeps longer than execution_timeout is canceled."""
    ref = durable_timeout_workflow.run_no_wait()

    with pytest.raises(Exception, match="(timeout|TIMED_OUT|failed)"):
        await ref.aio_result()


@pytest.mark.asyncio(loop_scope="session")
async def test_durable_execution_timeout_completes() -> None:
    """Task completes normally when within execution_timeout; evicted during sleep."""
    result: dict[str, Any] = await durable_timeout_completes_workflow.aio_run()
    out = result.get("durable_timeout_completes_task", result)
    assert out["status"] == "completed"


@pytest.mark.asyncio(loop_scope="session")
async def test_durable_refresh_timeout() -> None:
    """Task that would time out extends deadline via refresh_timeout and completes; evicted."""
    result: dict[str, Any] = await durable_refresh_timeout_workflow.aio_run()
    out = result.get("durable_refresh_timeout_task", result)
    assert out["status"] == "completed"


@pytest.mark.asyncio(loop_scope="session")
async def test_durable_timeout_eviction() -> None:
    """Evictable task: evicted during sleep, auto-restored, completes."""
    result: dict[str, Any] = await durable_timeout_eviction_workflow.aio_run()
    out = result.get("durable_timeout_eviction_task", result)
    assert out["status"] == "completed"
