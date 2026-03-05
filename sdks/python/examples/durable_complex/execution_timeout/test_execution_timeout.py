from __future__ import annotations

from typing import Any

import pytest

from examples.durable_complex.conftest import assert_evicted, requires_durable_eviction
from examples.durable_complex.execution_timeout.worker import (
    durable_refresh_timeout_workflow,
    durable_timeout_completes_workflow,
    durable_timeout_eviction_workflow,
    durable_timeout_workflow,
)
from hatchet_sdk import Hatchet


@pytest.mark.asyncio(loop_scope="session")
async def test_durable_execution_timeout_exceeded() -> None:
    """Durable task that sleeps longer than execution_timeout is canceled."""
    ref = durable_timeout_workflow.run_no_wait()

    with pytest.raises(Exception, match="(timeout|TIMED_OUT|failed)"):
        await ref.aio_result()


@requires_durable_eviction
@pytest.mark.asyncio(loop_scope="session")
async def test_durable_execution_timeout_completes(hatchet: Hatchet) -> None:
    """Task completes normally when within execution_timeout; evicted during sleep."""
    ref = durable_timeout_completes_workflow.run_no_wait()
    await assert_evicted(hatchet, ref.workflow_run_id)
    result: dict[str, Any] = await ref.aio_result()
    out = result.get("durable_timeout_completes_task", result)
    assert out["status"] == "completed"


@requires_durable_eviction
@pytest.mark.asyncio(loop_scope="session")
async def test_durable_refresh_timeout(hatchet: Hatchet) -> None:
    """Task that would time out extends deadline via refresh_timeout and completes; evicted."""
    ref = durable_refresh_timeout_workflow.run_no_wait()
    await assert_evicted(hatchet, ref.workflow_run_id)
    result: dict[str, Any] = await ref.aio_result()
    out = result.get("durable_refresh_timeout_task", result)
    assert out["status"] == "completed"


@requires_durable_eviction
@pytest.mark.asyncio(loop_scope="session")
async def test_durable_timeout_eviction(hatchet: Hatchet) -> None:
    """Evictable task: evicted during sleep, auto-restored, completes."""
    ref = durable_timeout_eviction_workflow.run_no_wait()
    await assert_evicted(hatchet, ref.workflow_run_id)
    result: dict[str, Any] = await ref.aio_result()
    out = result.get("durable_timeout_eviction_task", result)
    assert out["status"] == "completed"
