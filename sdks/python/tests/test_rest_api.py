import asyncio

import pytest

from examples.dag.worker import dag_workflow
from hatchet_sdk import Hatchet


@pytest.mark.asyncio(loop_scope="function")
async def test_list_runs(hatchet: Hatchet) -> None:
    dag_result = await dag_workflow.aio_run()

    runs = await hatchet.runs.aio_list(
        limit=10_000,
        only_tasks=True,
    )

    for v in dag_result.values():
        assert v in [r.output for r in runs.rows]


@pytest.mark.asyncio(loop_scope="function")
async def test_get_run(hatchet: Hatchet) -> None:
    dag_ref = await dag_workflow.aio_run_no_wait()

    await asyncio.sleep(3)

    run = await hatchet.runs.aio_get(dag_ref.workflow_run_id)

    assert run.run.display_name.startswith(dag_workflow.config.name)
    assert run.run.status.value == "COMPLETED"
    assert len(run.shape) == 4
    assert {t.name for t in dag_workflow.tasks} == {t.task_name for t in run.shape}


@pytest.mark.asyncio(loop_scope="function")
async def test_list_workflows(hatchet: Hatchet) -> None:
    workflows = await hatchet.workflows.aio_list(
        workflow_name=dag_workflow.config.name, limit=1, offset=0
    )

    assert workflows.rows
    assert len(workflows.rows) == 1

    workflow = workflows.rows[0]

    assert workflow.name == dag_workflow.config.name

    fetched_workflow = await hatchet.workflows.aio_get(workflow.metadata.id)

    assert fetched_workflow.name == dag_workflow.config.name
    assert fetched_workflow.metadata.id == workflow.metadata.id
