import asyncio

import pytest

from examples.cancellation.worker import cancellation_workflow
from hatchet_sdk import Hatchet
from hatchet_sdk.clients.rest.models.v1_task_status import V1TaskStatus
from examples.cancellation.worker import wf


@pytest.mark.asyncio()
async def test_cancellation(hatchet: Hatchet) -> None:
    ref = await cancellation_workflow.aio_run_no_wait()

    """Sleep for a long time since we only need cancellation to happen _eventually_"""
    await asyncio.sleep(10)

    for i in range(30):
        run = await hatchet.runs.aio_get(ref.workflow_run_id)

        if run.run.status == V1TaskStatus.RUNNING:
            await asyncio.sleep(1)
            continue

        assert run.run.status == V1TaskStatus.CANCELLED
        assert not run.run.output

        break
    else:
        assert False, "Workflow run did not cancel in time"

