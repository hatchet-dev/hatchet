from __future__ import annotations

from datetime import timedelta

from pydantic import BaseModel

from hatchet_sdk import DurableContext, EmptyModel, Hatchet
from hatchet_sdk.rate_limit import RateLimit, RateLimitDuration
from hatchet_sdk.runnables.eviction import EvictionPolicy

hatchet = Hatchet(debug=True)

RATE_LIMIT_KEY = "durable-complex-rate-limit"
RATE_LIMIT_DYNAMIC_PREFIX = "durable-rate-limit-dynamic"
SLEEP_SECONDS = 6
EVICTION_POLICY = EvictionPolicy(
    ttl=timedelta(seconds=1),
    allow_capacity_eviction=True,
    priority=0,
)


class DynamicRateLimitInput(BaseModel):
    group: str


durable_rate_limit_workflow = hatchet.workflow(name="DurableRateLimitWorkflow")


@durable_rate_limit_workflow.durable_task(
    execution_timeout=timedelta(minutes=5),
    rate_limits=[RateLimit(static_key=RATE_LIMIT_KEY, units=1)],
    eviction_policy=EVICTION_POLICY,
)
async def durable_rate_limit_task(
    input: EmptyModel, ctx: DurableContext
) -> dict[str, str]:
    await ctx.aio_sleep_for(timedelta(seconds=SLEEP_SECONDS))
    return {"status": "completed"}


durable_rate_limit_dynamic_workflow = hatchet.workflow(
    name="DurableRateLimitDynamicWorkflow",
    input_validator=DynamicRateLimitInput,
)


@durable_rate_limit_dynamic_workflow.durable_task(
    execution_timeout=timedelta(minutes=5),
    eviction_policy=EVICTION_POLICY,
    rate_limits=[
        RateLimit(
            dynamic_key="input.group",
            units=1,
            limit=2,
            duration=RateLimitDuration.SECOND,
        )
    ],
)
async def durable_rate_limit_dynamic_task(
    input: DynamicRateLimitInput, ctx: DurableContext
) -> dict[str, str]:
    await ctx.aio_sleep_for(timedelta(seconds=SLEEP_SECONDS))
    return {"status": "completed"}


def main() -> None:
    hatchet.rate_limits.put(RATE_LIMIT_KEY, 2, RateLimitDuration.SECOND)
    worker = hatchet.worker(
        "durable-complex-rate-limit-worker",
        workflows=[
            durable_rate_limit_workflow,
            durable_rate_limit_dynamic_workflow,
        ],
    )
    worker.start()


if __name__ == "__main__":
    main()
