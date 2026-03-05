from __future__ import annotations

from datetime import timedelta

from hatchet_sdk import DurableContext, EmptyModel, Hatchet
from hatchet_sdk.runnables.eviction import EvictionPolicy

hatchet = Hatchet(debug=True)

EVICTION_POLICY = EvictionPolicy(
    ttl=timedelta(seconds=1),
    allow_capacity_eviction=True,
    priority=0,
)

durable_timeout_workflow = hatchet.workflow(name="DurableExecutionTimeoutWorkflow")


@durable_timeout_workflow.durable_task(
    execution_timeout=timedelta(seconds=3),
)
async def durable_timeout_task(
    input: EmptyModel, ctx: DurableContext
) -> dict[str, str]:
    await ctx.aio_sleep_for(timedelta(seconds=10))
    return {"status": "completed"}


durable_timeout_completes_workflow = hatchet.workflow(
    name="DurableTimeoutCompletesWorkflow"
)


@durable_timeout_completes_workflow.durable_task(
    execution_timeout=timedelta(seconds=10),
    eviction_policy=EVICTION_POLICY,
)
async def durable_timeout_completes_task(
    input: EmptyModel, ctx: DurableContext
) -> dict[str, str]:
    await ctx.aio_sleep_for(timedelta(seconds=6))
    return {"status": "completed"}


durable_refresh_timeout_workflow = hatchet.workflow(
    name="DurableRefreshTimeoutWorkflow"
)


@durable_refresh_timeout_workflow.durable_task(
    execution_timeout=timedelta(seconds=4),
    eviction_policy=EVICTION_POLICY,
)
async def durable_refresh_timeout_task(
    input: EmptyModel, ctx: DurableContext
) -> dict[str, str]:
    ctx.refresh_timeout(timedelta(seconds=20))
    await ctx.aio_sleep_for(timedelta(seconds=6))
    return {"status": "completed"}


durable_timeout_eviction_workflow = hatchet.workflow(
    name="DurableExecutionTimeoutEvictionWorkflow"
)


@durable_timeout_eviction_workflow.durable_task(
    execution_timeout=timedelta(minutes=5),
    eviction_policy=EVICTION_POLICY,
)
async def durable_timeout_eviction_task(
    input: EmptyModel, ctx: DurableContext
) -> dict[str, str]:
    await ctx.aio_sleep_for(timedelta(seconds=6))
    return {"status": "completed"}


def main() -> None:
    worker = hatchet.worker(
        "durable-complex-timeout-worker",
        workflows=[
            durable_timeout_workflow,
            durable_timeout_completes_workflow,
            durable_refresh_timeout_workflow,
            durable_timeout_eviction_workflow,
        ],
    )
    worker.start()


if __name__ == "__main__":
    main()
