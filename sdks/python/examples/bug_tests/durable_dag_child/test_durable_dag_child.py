from examples.bug_tests.durable_dag_child.worker import parent_dag
import pytest
from hatchet_sdk import Hatchet, RunStatus, V1TaskStatus


@pytest.mark.asyncio(loop_scope="session")
async def test_durable_dag_child(hatchet: Hatchet) -> None:
    result = await parent_dag.aio_run(wait_for_result=False)
    await result.aio_result()

    details = await hatchet.runs.aio_get_details(result.workflow_run_id)

    assert details.status == RunStatus.COMPLETED

    assert {"step_a", "step_b"} == set(details.task_runs.keys())
    assert all(
        task.status == V1TaskStatus.COMPLETED for task in details.task_runs.values()
    )
