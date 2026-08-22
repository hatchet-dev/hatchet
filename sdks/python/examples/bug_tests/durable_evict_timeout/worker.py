from hatchet_sdk import Hatchet, EmptyModel, DurableContext, EvictionPolicy
from datetime import timedelta

hatchet = Hatchet()


@hatchet.durable_task(
    execution_timeout=timedelta(seconds=15),
    eviction_policy=EvictionPolicy(
        ttl=timedelta(seconds=1),
        allow_capacity_eviction=True,
    ),
)
async def evictable_durable(_i: EmptyModel, ctx: DurableContext) -> None:
    await ctx.aio_sleep_for(duration=timedelta(minutes=1))
