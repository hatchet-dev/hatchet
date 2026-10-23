import pytest

from hatchet_sdk import Hatchet, RunStatus
from datetime import timedelta

from examples.bug_tests.workflow_pause.worker import workflow_pause_concurrency_bug_task
import asyncio
import time


@pytest.mark.asyncio(loop_scope="session")
async def test_workflow_pause_under_concurrency(hatchet: Hatchet) -> None:
    await workflow_pause_concurrency_bug_task.aio_pause(
        queue_ttl=timedelta(seconds=60),
    )

    ref = await workflow_pause_concurrency_bug_task.aio_run(wait_for_result=False)

    start = time.time()

    while time.time() < start + 10:
        details = await hatchet.runs.aio_get_details(ref.workflow_run_id)

        assert (
            details.status == RunStatus.QUEUED
        ), f"Run {ref.workflow_run_id} is not queued."

        await asyncio.sleep(1)

    await hatchet.runs.aio_cancel(ref.workflow_run_id)
