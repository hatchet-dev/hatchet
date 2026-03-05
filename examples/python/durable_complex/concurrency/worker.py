from __future__ import annotations

from datetime import timedelta
from pydantic import BaseModel

from hatchet_sdk import (
    ConcurrencyExpression,
    ConcurrencyLimitStrategy,
    DurableContext,
    Hatchet,
)
from hatchet_sdk.runnables.eviction import EvictionPolicy

hatchet = Hatchet(debug=True)

SLEEP_SECONDS = 6
EVICTION_TTL_SECONDS = 1
EVICTION_POLICY = EvictionPolicy(
    ttl=timedelta(seconds=EVICTION_TTL_SECONDS),
    allow_capacity_eviction=True,
    priority=0,
)


class ConcurrencyInput(BaseModel):
    group: str


durable_concurrency_workflow = hatchet.workflow(
    name="DurableConcurrencyWorkflow",
    concurrency=ConcurrencyExpression(
        expression="input.group",
        max_runs=2,
        limit_strategy=ConcurrencyLimitStrategy.GROUP_ROUND_ROBIN,
    ),
    input_validator=ConcurrencyInput,
)


@durable_concurrency_workflow.durable_task(
    execution_timeout=timedelta(minutes=5),
    eviction_policy=EVICTION_POLICY,
)
async def durable_concurrency_task(
    input: ConcurrencyInput, ctx: DurableContext
) -> dict[str, str]:
    await ctx.aio_sleep_for(timedelta(seconds=SLEEP_SECONDS))
    return {"status": "completed", "group": input.group}


durable_concurrency_cancel_in_progress_workflow = hatchet.workflow(
    name="DurableConcurrencyCancelInProgress",
    concurrency=ConcurrencyExpression(
        expression="input.group",
        max_runs=1,
        limit_strategy=ConcurrencyLimitStrategy.CANCEL_IN_PROGRESS,
    ),
    input_validator=ConcurrencyInput,
)


@durable_concurrency_cancel_in_progress_workflow.durable_task(
    execution_timeout=timedelta(minutes=5),
    eviction_policy=EVICTION_POLICY,
)
async def durable_concurrency_cancel_in_progress_task(
    input: ConcurrencyInput, ctx: DurableContext
) -> dict[str, str]:
    await ctx.aio_sleep_for(timedelta(seconds=SLEEP_SECONDS))
    return {"status": "completed", "group": input.group}


durable_concurrency_cancel_newest_workflow = hatchet.workflow(
    name="DurableConcurrencyCancelNewest",
    concurrency=ConcurrencyExpression(
        expression="input.group",
        max_runs=1,
        limit_strategy=ConcurrencyLimitStrategy.CANCEL_NEWEST,
    ),
    input_validator=ConcurrencyInput,
)


@durable_concurrency_cancel_newest_workflow.durable_task(
    execution_timeout=timedelta(minutes=5),
    eviction_policy=EVICTION_POLICY,
)
async def durable_concurrency_cancel_newest_task(
    input: ConcurrencyInput, ctx: DurableContext
) -> dict[str, str]:
    await ctx.aio_sleep_for(timedelta(seconds=SLEEP_SECONDS))
    return {"status": "completed", "group": input.group}


durable_concurrency_slot_retention_workflow = hatchet.workflow(
    name="DurableConcurrencySlotRetention",
    concurrency=ConcurrencyExpression(
        expression="input.group",
        max_runs=1,
        limit_strategy=ConcurrencyLimitStrategy.GROUP_ROUND_ROBIN,
    ),
    input_validator=ConcurrencyInput,
)


@durable_concurrency_slot_retention_workflow.durable_task(
    execution_timeout=timedelta(minutes=5),
    eviction_policy=EVICTION_POLICY,
)
async def durable_concurrency_slot_retention_task(
    input: ConcurrencyInput, ctx: DurableContext
) -> dict[str, str]:
    await ctx.aio_sleep_for(timedelta(seconds=SLEEP_SECONDS))
    return {"status": "completed", "group": input.group}


def main() -> None:
    worker = hatchet.worker(
        "durable-complex-concurrency-worker",
        workflows=[
            durable_concurrency_workflow,
            durable_concurrency_cancel_in_progress_workflow,
            durable_concurrency_cancel_newest_workflow,
            durable_concurrency_slot_retention_workflow,
        ],
    )
    worker.start()


if __name__ == "__main__":
    main()
