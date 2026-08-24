import asyncio
import time

import pytest

from examples.bug_tests.durable_dag_child.worker import (
    dag_spawning_dag,
    diamond_dag,
    durable_spawner_dag,
    mixed_spawner_dag,
    multi_spawner_dag,
    parent_dag,
)
from hatchet_sdk import EmptyModel, Hatchet, RunStatus, V1TaskStatus
from hatchet_sdk.clients.admin import WorkflowRunDetail
from hatchet_sdk.runnables.workflow import Workflow

requires_durable_eviction = pytest.mark.usefixtures("_skip_unless_durable_eviction")


def spawned_run_ids(details: WorkflowRunDetail) -> set[str]:
    return {
        run_id
        for task in details.task_runs.values()
        for run_id in (task.output or {}).get("spawned_run_ids", [])
    }


async def wait_for_run(
    hatchet: Hatchet, run_id: str, timeout: float = 30.0
) -> WorkflowRunDetail:
    """Spawned children are fire-and-forget, so they outlive the run that spawned them."""
    deadline = time.monotonic() + timeout

    while True:
        details = await hatchet.runs.aio_get_details(run_id)

        if details.done:
            return details

        if time.monotonic() >= deadline:
            raise TimeoutError(
                f"run {run_id} was still {details.status} after {timeout}s"
            )

        await asyncio.sleep(0.5)


async def run_dag_to_completion(
    hatchet: Hatchet, workflow: Workflow[EmptyModel], expected_steps: set[str]
) -> WorkflowRunDetail:
    ref = await workflow.aio_run(wait_for_result=False)
    await ref.aio_result()

    details = await wait_for_run(hatchet, ref.workflow_run_id)

    assert details.status == RunStatus.COMPLETED
    assert set(details.task_runs.keys()) == expected_steps
    assert all(
        task.status == V1TaskStatus.COMPLETED for task in details.task_runs.values()
    )

    step_external_ids = {task.external_id for task in details.task_runs.values()}

    assert (
        not spawned_run_ids(details) & step_external_ids
    ), "a spawned child was deduped onto one of the DAG's own steps"

    return details


@pytest.mark.asyncio(loop_scope="session")
async def test_step_spawning_a_task(hatchet: Hatchet) -> None:
    await run_dag_to_completion(hatchet, parent_dag, {"step_a", "step_b"})


@pytest.mark.asyncio(loop_scope="session")
async def test_step_spawning_a_dag(hatchet: Hatchet) -> None:
    details = await run_dag_to_completion(
        hatchet, dag_spawning_dag, {"spawn_dag_step_a", "spawn_dag_step_b"}
    )

    child_run_ids = spawned_run_ids(details)

    assert len(child_run_ids) == 1

    child = await wait_for_run(hatchet, child_run_ids.pop())

    assert child.status == RunStatus.COMPLETED
    assert set(child.task_runs.keys()) == {"child_dag_step_a", "child_dag_step_b"}

    grandchild_run_ids = spawned_run_ids(child)

    assert len(grandchild_run_ids) == 1

    grandchild = await wait_for_run(hatchet, grandchild_run_ids.pop())

    assert grandchild.status == RunStatus.COMPLETED


@pytest.mark.asyncio(loop_scope="session")
async def test_root_claiming_every_step_index(hatchet: Hatchet) -> None:
    details = await run_dag_to_completion(
        hatchet,
        diamond_dag,
        {"diamond_root", "diamond_left", "diamond_right", "diamond_join"},
    )

    assert len(spawned_run_ids(details)) == 4


@requires_durable_eviction
@pytest.mark.asyncio(loop_scope="session")
async def test_durable_step_spawning_a_dag(hatchet: Hatchet) -> None:
    details = await run_dag_to_completion(
        hatchet,
        durable_spawner_dag,
        {"durable_spawner_first", "durable_spawner_second"},
    )

    child_run_ids = spawned_run_ids(details)

    assert len(child_run_ids) == 1

    child = await wait_for_run(hatchet, child_run_ids.pop())

    assert child.status == RunStatus.COMPLETED
    assert set(child.task_runs.keys()) == {"child_dag_step_a", "child_dag_step_b"}


@requires_durable_eviction
@pytest.mark.asyncio(loop_scope="session")
async def test_durable_and_regular_steps_both_spawning(hatchet: Hatchet) -> None:
    details = await run_dag_to_completion(
        hatchet,
        mixed_spawner_dag,
        {"mixed_durable_step", "mixed_regular_step", "mixed_final_step"},
    )

    assert len(spawned_run_ids(details)) == 2


@pytest.mark.asyncio(loop_scope="session")
async def test_multiple_steps_spawning(hatchet: Hatchet) -> None:
    details = await run_dag_to_completion(
        hatchet,
        multi_spawner_dag,
        {"multi_spawner_first", "multi_spawner_second", "multi_spawner_third"},
    )

    assert (
        len(spawned_run_ids(details)) == 2
    ), "each step's spawn must create its own child run"

    for readable_id in ("multi_spawner_first", "multi_spawner_second"):
        step = details.task_runs[readable_id]
        children = await hatchet.runs.aio_list(parent_task_external_id=step.external_id)

        assert {child.metadata.id for child in children.rows} == set(
            (step.output or {}).get("spawned_run_ids", [])
        ), f"{readable_id}'s spawned child must be attached to the step that spawned it"
