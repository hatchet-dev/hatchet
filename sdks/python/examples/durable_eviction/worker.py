"""
Minimal example demonstrating durable slot eviction.

The evictable_sleep task has a short eviction TTL (5s). When it enters a long
durable sleep, the eviction manager sees the TTL exceeded and evicts the task
-- freeing the worker slot.  A subsequent REST restore re-enqueues the task so
it can resume from its durable event log.
"""

from __future__ import annotations

import asyncio
from datetime import timedelta
from typing import Any

from hatchet_sdk import Context, DurableContext, EmptyModel, Hatchet, UserEventCondition
from hatchet_sdk.runnables.eviction import EvictionPolicy

hatchet = Hatchet(debug=True)


EVICTION_TTL_SECONDS = 5
LONG_SLEEP_SECONDS = 15
EVENT_KEY = "durable-eviction:event"

EVICTION_POLICY = EvictionPolicy(
    ttl=timedelta(seconds=EVICTION_TTL_SECONDS),
    allow_capacity_eviction=True,
    priority=0,
)


@hatchet.task()
async def child_task(input: EmptyModel, ctx: Context) -> dict[str, Any]:
    """Simple child that sleeps long enough for the parent's TTL to fire."""
    await asyncio.sleep(LONG_SLEEP_SECONDS)
    return {"child_status": "completed"}


@hatchet.durable_task(
    execution_timeout=timedelta(seconds=5),
    eviction_policy=EVICTION_POLICY,
)
async def evictable_sleep(input: EmptyModel, ctx: DurableContext) -> dict[str, Any]:
    """Sleeps long enough for the TTL-based eviction to kick in."""
    await ctx.aio_sleep_for(timedelta(seconds=LONG_SLEEP_SECONDS))
    return {"status": "completed"}


@hatchet.durable_task(
    execution_timeout=timedelta(minutes=5),
    eviction_policy=EVICTION_POLICY,
)
async def evictable_wait_for_event(
    input: EmptyModel, ctx: DurableContext
) -> dict[str, Any]:
    """Waits for a user event -- long enough for TTL eviction to fire."""
    await ctx.aio_wait_for(
        "event",
        UserEventCondition(event_key=EVENT_KEY, expression="true"),
    )
    return {"status": "completed"}


@hatchet.durable_task(
    execution_timeout=timedelta(minutes=5),
    eviction_policy=EVICTION_POLICY,
)
async def evictable_child_spawn(
    input: EmptyModel, ctx: DurableContext
) -> dict[str, Any]:
    """Spawns a child workflow whose runtime exceeds the eviction TTL."""
    child_result = await child_task.aio_run()
    return {"child": child_result, "status": "completed"}


@hatchet.durable_task(
    execution_timeout=timedelta(minutes=5),
    eviction_policy=EVICTION_POLICY,
)
async def multiple_eviction(input: EmptyModel, ctx: DurableContext) -> dict[str, Any]:
    """Sleeps twice, expecting eviction+restore after each sleep."""
    await ctx.aio_sleep_for(timedelta(seconds=LONG_SLEEP_SECONDS))
    await ctx.aio_sleep_for(timedelta(seconds=LONG_SLEEP_SECONDS))
    return {"status": "completed"}


@hatchet.durable_task(
    execution_timeout=timedelta(minutes=5),
    eviction_policy=EvictionPolicy(
        ttl=None,
        allow_capacity_eviction=False,
        priority=0,
    ),
)
async def non_evictable_sleep(input: EmptyModel, ctx: DurableContext) -> dict[str, Any]:
    """Has eviction disabled -- should never be evicted."""
    await ctx.aio_sleep_for(timedelta(seconds=10))
    return {"status": "completed"}


def main() -> None:
    worker = hatchet.worker(
        "eviction-worker",
        workflows=[
            evictable_sleep,
            evictable_wait_for_event,
            evictable_child_spawn,
            multiple_eviction,
            non_evictable_sleep,
            child_task,
        ],
    )
    worker.start()


if __name__ == "__main__":
    main()
