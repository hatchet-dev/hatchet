"""
Minimal example demonstrating durable slot eviction.

The evictable_sleep task has a short eviction TTL (5s). When it enters a long
durable sleep, the eviction manager sees the TTL exceeded and evicts the task
-- freeing the worker slot.  A subsequent REST restore re-enqueues the task so
it can resume from its durable event log.
"""

from __future__ import annotations

import time
from datetime import timedelta

from hatchet_sdk import DurableContext, EmptyModel, Hatchet
from hatchet_sdk.runnables.eviction import EvictionPolicy

hatchet = Hatchet(debug=True)


EVICTION_TTL_SECONDS = 5
LONG_SLEEP_SECONDS = 30


@hatchet.durable_task(
    execution_timeout=timedelta(minutes=5),
    eviction=EvictionPolicy(
        ttl=timedelta(seconds=EVICTION_TTL_SECONDS),
        allow_capacity_eviction=True,
        priority=0,
    ),
)
async def evictable_sleep(input: EmptyModel, ctx: DurableContext) -> dict[str, object]:
    """Sleeps long enough for the TTL-based eviction to kick in."""
    start = time.time()
    await ctx.aio_sleep_for(timedelta(seconds=LONG_SLEEP_SECONDS))
    return {"runtime": time.time() - start, "status": "completed"}


@hatchet.durable_task(
    execution_timeout=timedelta(minutes=5),
    eviction=EvictionPolicy(
        ttl=None,
        allow_capacity_eviction=False,
        priority=0,
    ),
)
async def non_evictable_sleep(
    input: EmptyModel, ctx: DurableContext
) -> dict[str, object]:
    """Has eviction disabled -- should never be evicted."""
    start = time.time()
    await ctx.aio_sleep_for(timedelta(seconds=10))
    return {"runtime": time.time() - start, "status": "completed"}


def main() -> None:
    worker = hatchet.worker(
        "eviction-worker",
        slots=2,
        workflows=[evictable_sleep, non_evictable_sleep],
    )
    worker.start()


if __name__ == "__main__":
    main()
