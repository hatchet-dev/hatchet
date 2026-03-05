from __future__ import annotations

import asyncio
import time
from typing import Any

import pytest

from examples.durable_complex.conftest import (
    assert_evicted,
    get_task_output,
    requires_durable_eviction,
)
from examples.durable_complex.retries.worker import (
    durable_retries_backoff_workflow,
    durable_retries_exhausted_workflow,
    durable_retries_sleep_workflow,
    durable_retries_workflow,
)
from hatchet_sdk import Hatchet
from hatchet_sdk.clients.rest.models.v1_task_status import V1TaskStatus

POLL_INTERVAL = 0.2
MAX_POLLS = 450


@pytest.mark.asyncio(loop_scope="session")
async def test_durable_retries() -> None:
    """Durable task with retries: fails once, succeeds on retry."""
    result: dict[str, Any] = await durable_retries_workflow.aio_run()

    out = get_task_output(
        result, "durable_retry_task", "durableretriesworkflow:durable_retry_task"
    )
    assert out.get("status") == "completed"
    assert out.get("retry_count") == 1


@pytest.mark.asyncio(loop_scope="session")
async def test_durable_retries_exhausted(hatchet: Hatchet) -> None:
    """Task with retries=2 that always fails reaches FAILED after 3 attempts."""
    ref = await durable_retries_exhausted_workflow.aio_run_no_wait()

    status = await hatchet.runs.aio_get_status(ref.workflow_run_id)
    for _ in range(MAX_POLLS):
        status = await hatchet.runs.aio_get_status(ref.workflow_run_id)
        if status == V1TaskStatus.FAILED:
            break
        await asyncio.sleep(POLL_INTERVAL)
    else:
        status = await hatchet.runs.aio_get_status(ref.workflow_run_id)

    assert status == V1TaskStatus.FAILED, f"Expected FAILED, got {status}"


@pytest.mark.asyncio(loop_scope="session")
async def test_durable_retries_backoff() -> None:
    """Task with backoff_factor succeeds after retries; timing between retries increases."""
    start = time.time()
    result: dict[str, Any] = await durable_retries_backoff_workflow.aio_run()
    elapsed = time.time() - start

    out = get_task_output(
        result,
        "durable_retry_backoff_task",
        "durableretriesbackoffworkflow:durable_retry_backoff_task",
    )
    assert out.get("status") == "completed"
    assert out.get("retry_count") == 3
    assert elapsed >= 2


@requires_durable_eviction
@pytest.mark.asyncio(loop_scope="session")
async def test_durable_retries_sleep(hatchet: Hatchet) -> None:
    """Durable task that fails after sleep succeeds on retry; evicted during sleep."""
    ref = durable_retries_sleep_workflow.run_no_wait()
    await assert_evicted(hatchet, ref.workflow_run_id)
    result: dict[str, Any] = await ref.aio_result()

    out = get_task_output(
        result,
        "durable_retry_sleep_task",
        "durableretriessleepworkflow:durable_retry_sleep_task",
    )
    assert out.get("status") == "completed"
    assert out.get("retry_count") == 1
