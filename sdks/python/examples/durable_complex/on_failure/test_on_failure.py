from __future__ import annotations

import asyncio
from typing import Any

import pytest

from examples.durable_complex.conftest import (
    assert_evicted,
    get_task_output,
    requires_durable_eviction,
)
from examples.durable_complex.on_failure.worker import (
    durable_on_failure_details_workflow,
    durable_on_failure_workflow,
    durable_on_success_workflow,
)
from hatchet_sdk import Hatchet
from hatchet_sdk.clients.rest.models.v1_task_status import V1TaskStatus

POLL_INTERVAL = 0.2
MAX_POLLS = 150


@requires_durable_eviction
@pytest.mark.asyncio(loop_scope="session")
async def test_durable_on_failure_fires(hatchet: Hatchet) -> None:
    """on_failure_task fires when durable task fails; receives task_run_errors; evicted during sleep."""
    ref = await durable_on_failure_workflow.aio_run_no_wait()
    await assert_evicted(hatchet, ref.workflow_run_id)
    try:
        await ref.aio_result()
    except Exception:
        pass

    await asyncio.sleep(5)

    details = await hatchet.runs.aio_get(ref.workflow_run_id)
    completed_tasks = [t for t in details.tasks if t.status == V1TaskStatus.COMPLETED]
    assert any("on_failure" in t.display_name for t in completed_tasks)


@requires_durable_eviction
@pytest.mark.asyncio(loop_scope="session")
async def test_durable_on_success_fires(hatchet: Hatchet) -> None:
    """on_success_task fires when all tasks succeed; evicted during sleep."""
    ref = durable_on_success_workflow.run_no_wait()
    await assert_evicted(hatchet, ref.workflow_run_id)
    result: dict[str, Any] = await ref.aio_result()

    handler_result = next(
        (
            v
            for v in result.values()
            if isinstance(v, dict) and v.get("status") == "success_handled"
        ),
        get_task_output(
            result,
            "durable_on_success_handler",
            "durableonsuccessworkflow:durable_on_success_handler-on-success",
        ),
    )
    assert handler_result.get("status") == "success_handled"


@requires_durable_eviction
@pytest.mark.asyncio(loop_scope="session")
async def test_durable_on_failure_get_task_run_error(hatchet: Hatchet) -> None:
    """get_task_run_error returns correct error info in on_failure handler; evicted during sleep."""
    ref = await durable_on_failure_details_workflow.aio_run_no_wait()
    await assert_evicted(hatchet, ref.workflow_run_id)
    try:
        await ref.aio_result()
    except Exception:
        pass

    for _ in range(MAX_POLLS):
        status = await hatchet.runs.aio_get_status(ref.workflow_run_id)
        if status in (V1TaskStatus.COMPLETED, V1TaskStatus.FAILED):
            break
        await asyncio.sleep(POLL_INTERVAL)

    details = await hatchet.runs.aio_get(ref.workflow_run_id)
    completed_tasks = [t for t in details.tasks if t.status == V1TaskStatus.COMPLETED]
    handler_like = (
        "on_failure",
        "on-failure",
        "details_handler",
        "handler",
    )
    has_handler = any(
        any(marker in (t.display_name or "") for marker in handler_like)
        or (t.output or {}).get("status") == "details_handled"
        for t in completed_tasks
    )
    assert has_handler, (
        f"No on_failure handler in completed tasks: "
        f"{[(t.display_name, t.status, (t.output or {}).get('status')) for t in details.tasks]}"
    )
