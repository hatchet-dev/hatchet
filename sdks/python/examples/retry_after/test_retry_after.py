import asyncio
import time

import pytest

from examples.retry_after.worker import (
    UpstreamInput,
    retry_after_exponential_backoff,
    retry_after_overrides_backoff,
    retry_after_upstream_delay,
)
from hatchet_sdk import Hatchet
from hatchet_sdk.clients.rest.api.task_api import TaskApi
from hatchet_sdk.clients.rest.models.v1_task_event_type import V1TaskEventType
from hatchet_sdk.exceptions import FailedTaskRunExceptionGroup


@pytest.mark.asyncio(loop_scope="session")
async def test_retry_after_waits_requested_delay(hatchet: Hatchet) -> None:
    start = time.monotonic()

    result = await retry_after_upstream_delay.aio_run(UpstreamInput(failing_attempts=2))

    elapsed = time.monotonic() - start

    assert result["attempt"] == 2
    assert elapsed >= 4


@pytest.mark.asyncio(loop_scope="session")
async def test_exponential_backoff_fails_when_retries_exhausted(
    hatchet: Hatchet,
) -> None:
    ref = await retry_after_exponential_backoff.aio_run(
        UpstreamInput(failing_attempts=100), wait_for_result=False
    )

    with pytest.raises(FailedTaskRunExceptionGroup):
        await ref.aio_result()

    await asyncio.sleep(3)

    task_run = await hatchet.runs.aio_get_task_run(ref.workflow_run_id)

    assert task_run.retry_count == 3


@pytest.mark.asyncio(loop_scope="session")
async def test_retry_after_overrides_configured_backoff(hatchet: Hatchet) -> None:
    ref = await retry_after_overrides_backoff.aio_run(
        UpstreamInput(failing_attempts=2), wait_for_result=False
    )

    result = await ref.aio_result()

    assert result["attempt"] == 2

    with hatchet.runs.client() as client:
        events = TaskApi(client).v1_task_event_list(task=ref.workflow_run_id, limit=100)

    retrying = [
        e.message for e in events.rows or [] if e.event_type == V1TaskEventType.RETRYING
    ]

    assert len(retrying) == 2
    assert all("The task requested a retry in" in m for m in retrying)

    run = await retry_after_overrides_backoff.aio_run(
        UpstreamInput(failing_attempts=100), wait_for_result=False
    )

    with pytest.raises(FailedTaskRunExceptionGroup):
        await run.aio_result()

    task_run = await hatchet.runs.aio_get_task_run(run.workflow_run_id)

    assert task_run.retry_count == 2
