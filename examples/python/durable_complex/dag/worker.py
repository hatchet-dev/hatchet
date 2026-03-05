from __future__ import annotations

from datetime import timedelta

from hatchet_sdk import Context, DurableContext, EmptyModel, Hatchet
from hatchet_sdk.runnables.eviction import EvictionPolicy

hatchet = Hatchet(debug=True)

SLEEP_SECONDS = 6
EVICTION_POLICY = EvictionPolicy(
    ttl=timedelta(seconds=1),
    allow_capacity_eviction=True,
    priority=0,
)

durable_dag_workflow = hatchet.workflow(name="DurableDAGWorkflow")


@durable_dag_workflow.task()
def ephemeral_parent(input: EmptyModel, ctx: Context) -> dict[str, str | int]:
    return {"value": 42, "status": "from_parent"}


@durable_dag_workflow.durable_task(
    parents=[ephemeral_parent],
    execution_timeout=timedelta(minutes=5),
    eviction_policy=EVICTION_POLICY,
)
async def durable_child(input: EmptyModel, ctx: DurableContext) -> dict[str, str | int]:
    parent_out = ctx.task_output(ephemeral_parent)
    await ctx.aio_sleep_for(timedelta(seconds=SLEEP_SECONDS))
    return {
        "status": "completed",
        "parent_value": parent_out["value"],
        "parent_status": parent_out["status"],
    }


durable_dag_durable_parent_workflow = hatchet.workflow(
    name="DurableDAGDurableParentWorkflow",
)


@durable_dag_durable_parent_workflow.durable_task(
    execution_timeout=timedelta(minutes=5),
    eviction_policy=EVICTION_POLICY,
)
async def durable_parent_task(
    input: EmptyModel, ctx: DurableContext
) -> dict[str, str | int]:
    await ctx.aio_sleep_for(timedelta(seconds=SLEEP_SECONDS))
    return {"value": 100, "status": "from_durable_parent"}


@durable_dag_durable_parent_workflow.durable_task(
    parents=[durable_parent_task],
    execution_timeout=timedelta(minutes=5),
    eviction_policy=EVICTION_POLICY,
)
async def durable_child_of_durable(
    input: EmptyModel, ctx: DurableContext
) -> dict[str, str | int]:
    parent_out = ctx.task_output(durable_parent_task)
    await ctx.aio_sleep_for(timedelta(seconds=SLEEP_SECONDS))
    return {
        "status": "completed",
        "parent_value": parent_out["value"],
        "parent_status": parent_out["status"],
    }


durable_dag_diamond_workflow = hatchet.workflow(name="DurableDAGDiamondWorkflow")


@durable_dag_diamond_workflow.task()
def diamond_a(input: EmptyModel, ctx: Context) -> dict[str, str]:
    return {"a": "done"}


@durable_dag_diamond_workflow.durable_task(
    parents=[diamond_a],
    execution_timeout=timedelta(minutes=5),
    eviction_policy=EVICTION_POLICY,
)
async def diamond_b(input: EmptyModel, ctx: DurableContext) -> dict[str, str]:
    a_out = ctx.task_output(diamond_a)
    await ctx.aio_sleep_for(timedelta(seconds=SLEEP_SECONDS))
    return {"b": "done", "from_a": a_out["a"]}


@durable_dag_diamond_workflow.durable_task(
    parents=[diamond_a],
    execution_timeout=timedelta(minutes=5),
    eviction_policy=EVICTION_POLICY,
)
async def diamond_c(input: EmptyModel, ctx: DurableContext) -> dict[str, str]:
    a_out = ctx.task_output(diamond_a)
    await ctx.aio_sleep_for(timedelta(seconds=SLEEP_SECONDS))
    return {"c": "done", "from_a": a_out["a"]}


@durable_dag_diamond_workflow.durable_task(
    parents=[diamond_b, diamond_c],
    execution_timeout=timedelta(minutes=5),
    eviction_policy=EVICTION_POLICY,
)
async def diamond_d(input: EmptyModel, ctx: DurableContext) -> dict[str, str]:
    b_out = ctx.task_output(diamond_b)
    c_out = ctx.task_output(diamond_c)
    await ctx.aio_sleep_for(timedelta(seconds=SLEEP_SECONDS))
    return {
        "status": "completed",
        "from_b": b_out["from_a"],
        "from_c": c_out["from_a"],
    }


durable_dag_parent_failure_workflow = hatchet.workflow(
    name="DurableDAGParentFailureWorkflow",
)


@durable_dag_parent_failure_workflow.task()
def failing_parent(input: EmptyModel, ctx: Context) -> dict[str, str]:
    raise RuntimeError("Parent fails for DAG test")


@durable_dag_parent_failure_workflow.durable_task(
    parents=[failing_parent],
    execution_timeout=timedelta(minutes=5),
    eviction_policy=EVICTION_POLICY,
)
async def durable_child_of_failing(
    input: EmptyModel, ctx: DurableContext
) -> dict[str, str | dict[str, str]]:
    parent_out = ctx.task_output(failing_parent)
    return {"status": "completed", "parent": parent_out}


def main() -> None:
    worker = hatchet.worker(
        "durable-complex-dag-worker",
        workflows=[
            durable_dag_workflow,
            durable_dag_durable_parent_workflow,
            durable_dag_diamond_workflow,
            durable_dag_parent_failure_workflow,
        ],
    )
    worker.start()


if __name__ == "__main__":
    main()
