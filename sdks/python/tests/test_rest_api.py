import asyncio

import pytest

from examples.dag.worker import dag_workflow
from hatchet_sdk import Hatchet


@pytest.mark.asyncio(loop_scope="session")
async def test_list_runs(hatchet: Hatchet) -> None:
    dag_result = await dag_workflow.aio_run()

    runs = await hatchet.runs.aio_list(
        limit=10_000,
        only_tasks=True,
    )

    for v in dag_result.values():
        assert v in [r.output for r in runs.rows]


@pytest.mark.asyncio(loop_scope="session")
async def test_get_run(hatchet: Hatchet) -> None:
    dag_ref = await dag_workflow.aio_run_no_wait()

    await asyncio.sleep(5)

    run = await hatchet.runs.aio_get(dag_ref.workflow_run_id)

    assert dag_workflow.config.name in run.run.display_name
    assert run.run.status.value == "COMPLETED"
    assert len(run.shape) == 4
    assert {t.name for t in dag_workflow.tasks} == {t.task_name for t in run.shape}


@pytest.mark.asyncio(loop_scope="session")
async def test_list_workflows(hatchet: Hatchet) -> None:
    workflows = await hatchet.workflows.aio_list(workflow_name=dag_workflow.config.name)

    assert workflows.rows
    assert len(workflows.rows) >= 1

    relevant_wf = next(
        iter(
            [
                wf
                for wf in workflows.rows
                if wf.name == hatchet.config.apply_namespace(dag_workflow.config.name)
            ]
        ),
        None,
    )

    assert relevant_wf is not None

    fetched_workflow = await hatchet.workflows.aio_get(relevant_wf.metadata.id)

    assert fetched_workflow.name == hatchet.config.apply_namespace(
        dag_workflow.config.name
    )
    assert fetched_workflow.metadata.id == relevant_wf.metadata.id
