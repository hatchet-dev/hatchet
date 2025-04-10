import asyncio

import pytest

from examples.on_failure.worker import on_failure_wf
from hatchet_sdk import Hatchet
from hatchet_sdk.clients.rest.models.v1_task_status import V1TaskStatus


@pytest.mark.asyncio(loop_scope="session")
async def test_run_timeout(hatchet: Hatchet) -> None:
    run = on_failure_wf.run_no_wait()
    try:
        await run.aio_result()

        assert False, "Expected workflow to timeout"
    except Exception as e:
        assert "step1 failed" in str(e)

    await asyncio.sleep(5)  # Wait for the on_failure job to finish

    details = await hatchet.runs.aio_get(run.workflow_run_id)

    assert len(details.tasks) == 2
    assert sum(t.status == V1TaskStatus.COMPLETED for t in details.tasks) == 1
    assert sum(t.status == V1TaskStatus.FAILED for t in details.tasks) == 1

    completed_task = next(
        t for t in details.tasks if t.status == V1TaskStatus.COMPLETED
    )
    failed_task = next(t for t in details.tasks if t.status == V1TaskStatus.FAILED)

    assert "on_failure" in completed_task.display_name
    assert "step1" in failed_task.display_name
