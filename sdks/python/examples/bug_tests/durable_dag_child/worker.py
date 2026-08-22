from pydantic import BaseModel

from hatchet_sdk import (
    Context,
    DurableContext,
    EmptyModel,
    Hatchet,
    TriggerWorkflowOptions,
)

hatchet = Hatchet()


class SpawnedRuns(BaseModel):
    spawned_run_ids: list[str]


@hatchet.task()
async def spawned_child(input: EmptyModel, ctx: Context) -> None:
    pass


spawned_child_dag = hatchet.workflow(name="spawned-child-dag")


@spawned_child_dag.task()
async def child_dag_step_a(input: EmptyModel, ctx: Context) -> None:
    pass


@spawned_child_dag.task(parents=[child_dag_step_a])
async def child_dag_step_b(input: EmptyModel, ctx: Context) -> SpawnedRuns:
    ref = await spawned_child.aio_run_no_wait()

    return SpawnedRuns(spawned_run_ids=[ref.workflow_run_id])


parent_dag = hatchet.workflow(name="colliding-dag")


@parent_dag.task()
async def step_a(input: EmptyModel, ctx: Context) -> SpawnedRuns:
    ref = await spawned_child.aio_run_no_wait()

    return SpawnedRuns(spawned_run_ids=[ref.workflow_run_id])


@parent_dag.task(parents=[step_a])
async def step_b(input: EmptyModel, ctx: Context) -> None:
    pass


dag_spawning_dag = hatchet.workflow(name="dag-spawning-dag")


@dag_spawning_dag.task()
async def spawn_dag_step_a(input: EmptyModel, ctx: Context) -> SpawnedRuns:
    ref = await spawned_child_dag.aio_run_no_wait()

    return SpawnedRuns(spawned_run_ids=[ref.workflow_run_id])


@dag_spawning_dag.task(parents=[spawn_dag_step_a])
async def spawn_dag_step_b(input: EmptyModel, ctx: Context) -> None:
    pass


diamond_dag = hatchet.workflow(name="diamond-spawning-dag")

DIAMOND_STEP_COUNT = 4


@diamond_dag.task()
async def diamond_root(input: EmptyModel, ctx: Context) -> SpawnedRuns:
    # claim every step index in the DAG, not just the next unused spawn index
    refs = [
        await spawned_child.aio_run_no_wait(
            options=TriggerWorkflowOptions(child_index=i)
        )
        for i in range(DIAMOND_STEP_COUNT)
    ]

    return SpawnedRuns(spawned_run_ids=[ref.workflow_run_id for ref in refs])


@diamond_dag.task(parents=[diamond_root])
async def diamond_left(input: EmptyModel, ctx: Context) -> None:
    pass


@diamond_dag.task(parents=[diamond_root])
async def diamond_right(input: EmptyModel, ctx: Context) -> None:
    pass


@diamond_dag.task(parents=[diamond_left, diamond_right])
async def diamond_join(input: EmptyModel, ctx: Context) -> None:
    pass


multi_spawner_dag = hatchet.workflow(name="multi-spawner-dag")


@multi_spawner_dag.task()
async def multi_spawner_first(input: EmptyModel, ctx: Context) -> SpawnedRuns:
    ref = await spawned_child.aio_run_no_wait()

    return SpawnedRuns(spawned_run_ids=[ref.workflow_run_id])


@multi_spawner_dag.task(parents=[multi_spawner_first])
async def multi_spawner_second(input: EmptyModel, ctx: Context) -> SpawnedRuns:
    ref = await spawned_child.aio_run_no_wait()

    return SpawnedRuns(spawned_run_ids=[ref.workflow_run_id])


@multi_spawner_dag.task(parents=[multi_spawner_second])
async def multi_spawner_third(input: EmptyModel, ctx: Context) -> None:
    pass


durable_spawner_dag = hatchet.workflow(name="durable-spawner-dag")


@durable_spawner_dag.durable_task()
async def durable_spawner_first(input: EmptyModel, ctx: DurableContext) -> SpawnedRuns:
    # inside a durable context this routes through the durable event log rather than the
    # admin trigger path, so the dedupe row lands on this step instead of the orchestrator
    ref = await spawned_child_dag.aio_run_no_wait()

    return SpawnedRuns(spawned_run_ids=[ref.workflow_run_id])


@durable_spawner_dag.task(parents=[durable_spawner_first])
async def durable_spawner_second(input: EmptyModel, ctx: Context) -> None:
    pass


mixed_spawner_dag = hatchet.workflow(name="mixed-spawner-dag")


@mixed_spawner_dag.durable_task()
async def mixed_durable_step(input: EmptyModel, ctx: DurableContext) -> SpawnedRuns:
    ref = await spawned_child.aio_run_no_wait()

    return SpawnedRuns(spawned_run_ids=[ref.workflow_run_id])


@mixed_spawner_dag.task(parents=[mixed_durable_step])
async def mixed_regular_step(input: EmptyModel, ctx: Context) -> SpawnedRuns:
    ref = await spawned_child.aio_run_no_wait()

    return SpawnedRuns(spawned_run_ids=[ref.workflow_run_id])


@mixed_spawner_dag.task(parents=[mixed_regular_step])
async def mixed_final_step(input: EmptyModel, ctx: Context) -> None:
    pass
