"""
Minimal worker for durable and durable eviction load tests.
"""

from __future__ import annotations

import asyncio
from datetime import timedelta
from typing import Any

from hatchet_sdk import Context, DurableContext, EmptyModel, Hatchet, UserEventCondition
from hatchet_sdk.runnables.eviction import EvictionPolicy

hatchet = Hatchet(debug=True)

LOAD_EVENT_KEY = "loadtest:event"
EVICTION_TTL_SECONDS = 5
SLEEP_SECONDS = 1
LONG_SLEEP_SECONDS = 10

EVICTION_POLICY = EvictionPolicy(
    ttl=timedelta(seconds=EVICTION_TTL_SECONDS),
    allow_capacity_eviction=True,
    priority=0,
)


@hatchet.task()
async def load_child_task(input: EmptyModel, ctx: Context) -> dict[str, Any]:
    await asyncio.sleep(2)
    return {"status": "done"}


@hatchet.durable_task(execution_timeout=timedelta(minutes=5))
async def load_durable_sleep(input: EmptyModel, ctx: DurableContext) -> dict[str, Any]:
    await ctx.aio_sleep_for(timedelta(seconds=SLEEP_SECONDS))
    return {"status": "completed"}


@hatchet.durable_task(execution_timeout=timedelta(minutes=5))
async def load_durable_sleep_event(
    input: EmptyModel, ctx: DurableContext
) -> dict[str, Any]:
    await ctx.aio_sleep_for(timedelta(seconds=SLEEP_SECONDS))
    await ctx.aio_wait_for(
        "event",
        UserEventCondition(event_key=LOAD_EVENT_KEY, expression="true"),
    )
    return {"status": "completed"}


@hatchet.durable_task(execution_timeout=timedelta(minutes=5))
async def load_durable_child_spawn(
    input: EmptyModel, ctx: DurableContext
) -> dict[str, Any]:
    result = await load_child_task.aio_run()
    return {"child": result, "status": "completed"}


@hatchet.durable_task(
    execution_timeout=timedelta(minutes=5),
    eviction_policy=EVICTION_POLICY,
)
async def load_evictable_sleep(
    input: EmptyModel, ctx: DurableContext
) -> dict[str, Any]:
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
async def load_non_evictable_sleep(
    input: EmptyModel, ctx: DurableContext
) -> dict[str, Any]:
    await ctx.aio_sleep_for(timedelta(seconds=5))
    return {"status": "completed"}


def main() -> None:
    worker = hatchet.worker(
        "load-test-worker",
        workflows=[
            load_durable_sleep,
            load_durable_sleep_event,
            load_durable_child_spawn,
            load_evictable_sleep,
            load_non_evictable_sleep,
            load_child_task,
        ],
    )
    worker.start()


if __name__ == "__main__":
    main()
