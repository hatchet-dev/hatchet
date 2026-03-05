from __future__ import annotations

import asyncio
from typing import Any

import pytest

from hatchet_sdk import Hatchet
from hatchet_sdk.clients.rest.models.v1_task_status import V1TaskStatus

POLL_INTERVAL = 0.2
MAX_EVICTION_POLLS = 150

requires_durable_eviction = pytest.mark.usefixtures("_skip_unless_durable_eviction")


def get_task_output(result: dict[str, Any], *preferred_keys: str) -> dict[str, Any]:
    """
    Extract task output from workflow result.

    Result keys may be task_name or workflowname:taskname. Falls back to
    first dict value when preferred keys are not found.
    """
    for key in preferred_keys:
        out = result.get(key)
        if out is not None and isinstance(out, dict):
            return dict(out)
    fallback: dict[str, Any] = next(
        (v for v in result.values() if isinstance(v, dict)), {}
    )
    return fallback


async def assert_evicted(hatchet: Hatchet, workflow_run_id: str) -> None:
    """Poll until at least one task in the workflow run reaches EVICTED status."""
    for _ in range(MAX_EVICTION_POLLS):
        details = await hatchet.runs.aio_get_details(workflow_run_id)
        if any(t.status == V1TaskStatus.EVICTED for t in details.task_runs.values()):
            return
        await asyncio.sleep(POLL_INTERVAL)

    details = await hatchet.runs.aio_get_details(workflow_run_id)
    statuses = {t.status for t in details.task_runs.values()}
    raise AssertionError(f"Expected EVICTED status but got: {statuses}")
