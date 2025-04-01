import asyncio

import pytest

from examples.cancellation.worker import cancellation_workflow
from hatchet_sdk import Hatchet
from hatchet_sdk.clients.rest.models.v1_task_status import V1TaskStatus


@pytest.mark.asyncio()
async def test_cancellation(hatchet: Hatchet) -> None:
    ref = await cancellation_workflow.aio_run_no_wait()

    """Sleep for a long time since we only need cancellation to happen _eventually_"""
    await asyncio.sleep(10)

    run = await hatchet.runs.aio_get(ref.workflow_run_id)

    assert run.run.status == V1TaskStatus.CANCELLED
    assert not run.run.output
