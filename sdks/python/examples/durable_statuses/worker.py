from __future__ import annotations

from datetime import timedelta
from typing import Any

from hatchet_sdk import DurableContext, EmptyModel, Hatchet

hatchet = Hatchet(debug=True)


@hatchet.durable_task(execution_timeout=timedelta(seconds=10))
async def status_short_sleep(input: EmptyModel, ctx: DurableContext) -> dict[str, Any]:
    """Short sleep for quick COMPLETED status checks."""
    await ctx.aio_sleep_for(timedelta(seconds=1))
    return {"status": "completed"}


@hatchet.durable_task(execution_timeout=timedelta(seconds=30))
async def status_long_sleep(input: EmptyModel, ctx: DurableContext) -> dict[str, Any]:
    """Long sleep for QUEUED/RUNNING status checks before completion."""
    await ctx.aio_sleep_for(timedelta(seconds=20))
    return {"status": "completed"}


def main() -> None:
    from examples.durable.worker import durable_error_task
    from examples.durable_eviction.worker import (
        evictable_sleep,
        evictable_wait_for_event,
    )

    worker = hatchet.worker(
        "durable-statuses-worker",
        workflows=[
            status_short_sleep,
            status_long_sleep,
            durable_error_task,
            evictable_sleep,
            evictable_wait_for_event,
        ],
    )
    worker.start()


if __name__ == "__main__":
    main()
