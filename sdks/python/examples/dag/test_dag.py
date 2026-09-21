import pytest
import tenacity
from tenacity import stop_after_attempt, wait_exponential

from examples.dag.worker import dag_workflow, DAGWorkflowInput
from hatchet_sdk import Hatchet
from hatchet_sdk.clients.rest.models.v1_task_status import V1TaskStatus
from hatchet_sdk.clients.rest.models.v1_task_summary import V1TaskSummary


@pytest.mark.asyncio(loop_scope="session")
async def test_run(hatchet: Hatchet) -> None:
    result = await dag_workflow.aio_run()

    one = result["step1"]["random_number"]
    two = result["step2"]["random_number"]
    assert result["step3"]["sum"] == one + two
    assert result["step4"]["step4"] == "step4"


@tenacity.retry(
    stop=stop_after_attempt(10), wait=wait_exponential(multiplier=1, min=1, max=5)
)
async def get_on_failure_task(hatchet: Hatchet, run_id: str) -> V1TaskSummary:
    details = await hatchet.runs.aio_get(run_id)
    task = next((t for t in details.tasks if "on_failure" in t.display_name), None)
    if task is None:
        raise Exception(
            f"on-failure task was never created; tasks seen so far: "
            f"{[t.display_name for t in details.tasks]}"
        )

    return task


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

    assert on_failure_task.status == V1TaskStatus.COMPLETED
    assert on_failure_task.output == {}


@pytest.mark.asyncio(loop_scope="session")
async def test_on_failure_task_is_runs_on_failure(hatchet: Hatchet) -> None:
    ref = dag_workflow.run(
        input=DAGWorkflowInput(should_fail=True),
        wait_for_result=False,
    )

    with pytest.raises(Exception):
        await ref.aio_result()

    on_failure_task = await get_on_failure_task(
        hatchet=hatchet, run_id=ref.workflow_run_id
    )

    assert on_failure_task.status == V1TaskStatus.COMPLETED
    assert on_failure_task.output == {"ran": True}
