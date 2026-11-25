import pytest

from examples.cancellation.worker import cancellation_workflow
from examples.test_utils import wait_for_run_done
from hatchet_sdk import Hatchet


@pytest.mark.asyncio(loop_scope="session")
async def test_cancellation(hatchet: Hatchet) -> None:
    ref = await cancellation_workflow.aio_run(wait_for_result=False)

    run = await wait_for_run_done(hatchet, ref.workflow_run_id, timeout=120)

    assert run.status == "CANCELLED"
