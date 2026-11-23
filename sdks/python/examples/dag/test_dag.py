import pytest

from examples.dag.worker import DAGWorkflowInput, dag_workflow
from examples.test_utils import poll_until
from hatchet_sdk import FailedTaskRunExceptionGroup, Hatchet
from hatchet_sdk.clients.rest.models.v1_task_summary import V1TaskSummary
from hatchet_sdk.clients.rest.models.v1_workflow_run_details import V1WorkflowRunDetails


@pytest.mark.asyncio(loop_scope="session")
async def test_run(hatchet: Hatchet) -> None:
    result = await dag_workflow.aio_run()

    one = result["step1"]["random_number"]
    two = result["step2"]["random_number"]
    assert result["step3"]["sum"] == one + two
    assert result["step4"]["step4"] == "step4"


async def get_on_failure_task(hatchet: Hatchet, run_id: str) -> V1TaskSummary:
    def find_on_failure_task(details: V1WorkflowRunDetails) -> V1TaskSummary | None:
        return next((t for t in details.tasks if "on_failure" in t.display_name), None)

    details = await poll_until(
        lambda: hatchet.runs.aio_get(run_id),
        lambda details: (task := find_on_failure_task(details)) is not None
        and task.status not in ("QUEUED", "RUNNING"),
        description=f"the on-failure task of run {run_id} to finish",
        timeout=30.0,
        interval=1.0,
    )

    on_failure_task = find_on_failure_task(details)
    assert on_failure_task is not None
    return on_failure_task


@pytest.mark.asyncio(loop_scope="session")
async def test_on_failure_task_is_skipped_not_stuck(hatchet: Hatchet) -> None:
    ref = dag_workflow.run(wait_for_result=False)

    result = await ref.aio_result()
    one = result["step1"]["random_number"]
    two = result["step2"]["random_number"]

    assert result["step3"]["sum"] == one + two

    on_failure_task = await get_on_failure_task(
        hatchet=hatchet, run_id=ref.workflow_run_id
    )

    assert on_failure_task.status == "COMPLETED"
    assert on_failure_task.output == {}


@pytest.mark.asyncio(loop_scope="session")
async def test_on_failure_task_is_runs_on_failure(hatchet: Hatchet) -> None:
    ref = dag_workflow.run(
        input=DAGWorkflowInput(should_fail=True),
        wait_for_result=False,
    )

    with pytest.raises(FailedTaskRunExceptionGroup):
        await ref.aio_result()

    on_failure_task = await get_on_failure_task(
        hatchet=hatchet, run_id=ref.workflow_run_id
    )

    assert on_failure_task.status == "COMPLETED"
    assert on_failure_task.output == {"ran": True}
