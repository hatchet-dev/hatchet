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

durable_retries_workflow = hatchet.workflow(name="DurableRetriesWorkflow")


@durable_retries_workflow.durable_task(
    execution_timeout=timedelta(minutes=5),
    retries=2,
)
async def durable_retry_task(
    input: EmptyModel, ctx: DurableContext
) -> dict[str, str | int]:
    if ctx.retry_count < 1:
        raise RuntimeError("Intentional failure for retry test")
    return {"status": "completed", "retry_count": ctx.retry_count}


durable_retries_exhausted_workflow = hatchet.workflow(
    name="DurableRetriesExhaustedWorkflow"
)


@durable_retries_exhausted_workflow.durable_task(
    execution_timeout=timedelta(minutes=5),
    retries=2,
)
async def durable_retry_exhausted_task(
    input: EmptyModel, ctx: DurableContext
) -> dict[str, str]:
    raise RuntimeError("Always fails for retry exhausted test")


durable_retries_backoff_workflow = hatchet.workflow(
    name="DurableRetriesBackoffWorkflow"
)


@durable_retries_backoff_workflow.durable_task(
    execution_timeout=timedelta(minutes=5),
    retries=3,
    backoff_factor=2.0,
    backoff_max_seconds=10,
)
async def durable_retry_backoff_task(
    input: EmptyModel, ctx: DurableContext
) -> dict[str, str | int]:
    if ctx.retry_count < 3:
        raise RuntimeError("Intentional failure for backoff test")
    return {"status": "completed", "retry_count": ctx.retry_count}


durable_retries_sleep_workflow = hatchet.workflow(name="DurableRetriesSleepWorkflow")


@durable_retries_sleep_workflow.durable_task(
    execution_timeout=timedelta(minutes=5),
    retries=2,
    eviction_policy=EVICTION_POLICY,
)
async def durable_retry_sleep_task(
    input: EmptyModel, ctx: DurableContext
) -> dict[str, str | int]:
    await ctx.aio_sleep_for(timedelta(seconds=6))
    if ctx.retry_count < 1:
        raise RuntimeError("Intentional failure after sleep for retry test")
    return {"status": "completed", "retry_count": ctx.retry_count}


def main() -> None:
    worker = hatchet.worker(
        "durable-complex-retries-worker",
        workflows=[
            durable_retries_workflow,
            durable_retries_exhausted_workflow,
            durable_retries_backoff_workflow,
            durable_retries_sleep_workflow,
        ],
    )
    worker.start()


if __name__ == "__main__":
    main()
