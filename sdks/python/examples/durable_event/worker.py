from __future__ import annotations

from datetime import timedelta
from typing import Any

from hatchet_sdk import DurableContext, EmptyModel, Hatchet, UserEventCondition
from hatchet_sdk.runnables.eviction import EvictionPolicy

hatchet = Hatchet(debug=True)

EVENT_KEY = "user:update"
EVICTION_POLICY = EvictionPolicy(
    ttl=timedelta(seconds=1),
    allow_capacity_eviction=True,
    priority=0,
)


# > Durable Event
@hatchet.durable_task(
    name="DurableEventTask",
    eviction_policy=EVICTION_POLICY,
)
async def durable_event_task(input: EmptyModel, ctx: DurableContext) -> dict[str, Any]:
    res = await ctx.aio_wait_for(
        "event",
        UserEventCondition(event_key="user:update"),
    )
    print("got event", res)
    return res


# !!


@hatchet.durable_task(
    name="DurableEventWithFilterTask",
    eviction_policy=EVICTION_POLICY,
)
async def durable_event_task_with_filter(
    input: EmptyModel, ctx: DurableContext
) -> dict[str, Any]:
    # > Durable Event With Filter
    res = await ctx.aio_wait_for(
        "event",
        UserEventCondition(
            event_key="user:update", expression="input.user_id == '1234'"
        ),
    )
    # !!
    print("got event", res)
    return res


@hatchet.durable_task(
    name="DurableEventWithFilterMismatchTask",
    eviction_policy=EVICTION_POLICY,
)
async def durable_event_task_filter_mismatch(
    input: EmptyModel, ctx: DurableContext
) -> dict[str, Any]:
    res = await ctx.aio_wait_for(
        "event",
        UserEventCondition(
            event_key="user:update", expression="input.user_id == '9999'"
        ),
    )
    print("got event", res)
    return res


def main() -> None:
    worker = hatchet.worker(
        "durable-event-worker",
        workflows=[
            durable_event_task,
            durable_event_task_with_filter,
            durable_event_task_filter_mismatch,
        ],
    )
    worker.start()


if __name__ == "__main__":
    main()
