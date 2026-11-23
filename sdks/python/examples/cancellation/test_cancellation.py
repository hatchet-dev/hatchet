import asyncio

import pytest

from examples.cancellation.worker import cancellation_workflow
from hatchet_sdk import Hatchet


@pytest.mark.asyncio(loop_scope="session")
async def test_cancellation(hatchet: Hatchet) -> None:
    ref = await cancellation_workflow.aio_run(wait_for_result=False)

    for _ in range(120):
        run = await hatchet.runs.aio_get_details(ref.workflow_run_id)

        if run.status in ["RUNNING", "QUEUED"]:
            await asyncio.sleep(1)
            continue

        assert run.status == "CANCELLED"

        break
    else:
        pytest.fail("Workflow run did not cancel in time")
