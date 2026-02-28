from __future__ import annotations

from datetime import timedelta
from typing import Any

from hatchet_sdk import DurableContext, EmptyModel, Hatchet
from hatchet_sdk.runnables.eviction import EvictionPolicy

hatchet = Hatchet(debug=True)


# > Durable Sleep
SLEEP_DURATION = 6
EVICTION_POLICY = EvictionPolicy(
    ttl=timedelta(seconds=1),
    allow_capacity_eviction=True,
    priority=0,
)


@hatchet.durable_task(
    name="DurableSleepTask",
    eviction_policy=EVICTION_POLICY,
)
async def durable_sleep_task(input: EmptyModel, ctx: DurableContext) -> dict[str, Any]:
    res = await ctx.aio_sleep_for(timedelta(seconds=SLEEP_DURATION))
    print("got result", res)
    return res


# !!


def main() -> None:
    worker = hatchet.worker("durable-sleep-worker", workflows=[durable_sleep_task])
    worker.start()


if __name__ == "__main__":
    main()
