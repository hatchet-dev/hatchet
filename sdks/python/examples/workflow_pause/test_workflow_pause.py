import pytest

from hatchet_sdk import Hatchet, RunStatus
from datetime import timedelta

from examples.simple.worker import simple
import asyncio
import time


@pytest.mark.asyncio(loop_scope="session")
async def test_workflow_pause_cancel_after_ttl(hatchet: Hatchet) -> None:
    await simple.aio_pause(
        queue_ttl=timedelta(seconds=1),
    )

    ref = await simple.aio_run(wait_for_result=False)

    start_time = time.time()
    run_id = ref.workflow_run_id
    timeout = 10  # seconds

    while True:
        details = await hatchet.runs.aio_get_details(run_id)

        if details.status == RunStatus.CANCELLED:
            return

        assert details.status == RunStatus.QUEUED, f"Run {run_id} is not queued."

        if time.time() - start_time > timeout:
            assert False, f"Run {run_id} was not cancelled within {timeout} seconds."

        await asyncio.sleep(1)


@pytest.mark.asyncio(loop_scope="session")
async def test_workflow_unpause(hatchet: Hatchet) -> None:
    await simple.aio_pause(
        queue_ttl=timedelta(minutes=10),
    )

    ref = await simple.aio_run(wait_for_result=False)
    run_id = ref.workflow_run_id

    for _ in range(3):
        details = await hatchet.runs.aio_get_details(run_id)

        assert details.status == RunStatus.QUEUED, f"Run {run_id} is not queued."

        await asyncio.sleep(1)

    await simple.aio_unpause()

    timeout = 60  # seconds - this can take a while because we rely on polling internally to re-queue
    start = time.time()

    while True:
        if time.time() - start > timeout:
            assert False, f"Run {run_id} was not completed within {timeout} seconds."

        details = await hatchet.runs.aio_get_details(run_id)

        if details.status == RunStatus.CANCELLED:
            assert False, f"Run {run_id} was cancelled after unpausing."

        if details.status == RunStatus.COMPLETED:
            return

        assert details.status in [
            RunStatus.QUEUED,
            RunStatus.RUNNING,
        ], f"Run {run_id} is not queued or running."

        await asyncio.sleep(1)
