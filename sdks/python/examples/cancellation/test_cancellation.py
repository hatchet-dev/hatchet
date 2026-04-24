import asyncio

import pytest

from examples.cancellation.worker import cancellation_workflow
from hatchet_sdk import Hatchet, RunStatus
from hatchet_sdk.clients.rest.models.v1_task_status import V1TaskStatus


@pytest.mark.asyncio(loop_scope="session")
async def test_cancellation(hatchet: Hatchet) -> None:
    ref = await cancellation_workflow.aio_run(wait_for_result=False)

    for _ in range(30):
        run = await hatchet.runs.aio_get_details(ref.workflow_run_id)

        if run.status in [RunStatus.RUNNING, RunStatus.QUEUED]:
            await asyncio.sleep(1)
            continue

        assert run.status == RunStatus.CANCELLED

        break
    else:
        assert False, "Workflow run did not cancel in time"
