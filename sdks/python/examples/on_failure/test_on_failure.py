import pytest
import tenacity
from tenacity import stop_after_attempt, wait_exponential

from examples.on_failure.worker import (
    ERROR_TEXT,
    details_on_failure,
    details_step1,
    on_failure_wf,
    on_failure_wf_with_details,
)
from hatchet_sdk import Hatchet, V1TaskStatus
from hatchet_sdk.clients.admin import WorkflowRunDetail


@pytest.mark.asyncio(loop_scope="session")
async def test_run_timeout(hatchet: Hatchet) -> None:
    run = on_failure_wf.run(wait_for_result=False)
    try:
        await run.aio_result()

        assert False, "Expected workflow to timeout"
    except Exception as e:
        assert "step1 failed" in str(e)

    @tenacity.retry(
        stop=stop_after_attempt(5), wait=wait_exponential(multiplier=1, min=4, max=10)
    )
    async def get_runs() -> WorkflowRunDetail:
        details = await hatchet.runs.aio_get_details(run.workflow_run_id)
        if len(details.task_runs) == 2 and all(
            t.status in [V1TaskStatus.COMPLETED, V1TaskStatus.FAILED]
            for t in details.task_runs.values()
        ):
            return details
        raise Exception()

    details = await get_runs()
    assert len(details.task_runs) == 2
    assert (
        sum(t.status == V1TaskStatus.COMPLETED for t in details.task_runs.values()) == 1
    )
    assert sum(t.status == V1TaskStatus.FAILED for t in details.task_runs.values()) == 1

    completed_task = next(
        t for t in details.task_runs.values() if t.status == V1TaskStatus.COMPLETED
    )
    failed_task = next(
        t for t in details.task_runs.values() if t.status == V1TaskStatus.FAILED
    )

    assert "on_failure" in completed_task.readable_id
    assert "step1" in failed_task.readable_id


@pytest.mark.asyncio(loop_scope="session")
async def test_on_failure_task_receives_upstream_errors(hatchet: Hatchet) -> None:
    run = on_failure_wf_with_details.run(wait_for_result=False)

    with pytest.raises(Exception, match=ERROR_TEXT):
        await run.aio_result()

    @tenacity.retry(
        stop=stop_after_attempt(5), wait=wait_exponential(multiplier=1, min=4, max=10)
    )
    async def get_runs() -> WorkflowRunDetail:
        details = await hatchet.runs.aio_get_details(run.workflow_run_id)
        if len(details.task_runs) == 2 and all(
            t.status in [V1TaskStatus.COMPLETED, V1TaskStatus.FAILED]
            for t in details.task_runs.values()
        ):
            return details
        raise Exception()

    details = await get_runs()

    failed_task = next(
        t for t in details.task_runs.values() if t.status == V1TaskStatus.FAILED
    )
    on_failure_task = next(
        t for t in details.task_runs.values() if t.status == V1TaskStatus.COMPLETED
    )

    assert details_step1.name in failed_task.readable_id
    assert details_on_failure.name in on_failure_task.readable_id

    assert on_failure_task.output == {
        "status": "success",
        "failed_run_external_id": failed_task.external_id,
    }
