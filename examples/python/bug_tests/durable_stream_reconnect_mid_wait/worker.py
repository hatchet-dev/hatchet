import asyncio
import time
from contextlib import suppress
from datetime import timedelta
from typing import Any

from hatchet_sdk import DurableContext, Hatchet

hatchet = Hatchet()


async def reconnect_durable_stream(listener: Any, after_seconds: float) -> None:
    await asyncio.sleep(after_seconds)
    listener._receive_task.cancel()
    with suppress(asyncio.CancelledError):
        await listener._receive_task
    await listener._connect()
    listener._receive_task = asyncio.create_task(listener._receive_loop())


@hatchet.durable_task(execution_timeout=timedelta(minutes=3))
async def durable_stream_reconnect_mid_wait(
    _i: None, ctx: DurableContext
) -> dict[str, float]:
    await ctx.aio_sleep_for(timedelta(seconds=1))

    reconnect = asyncio.create_task(
        reconnect_durable_stream(ctx._durable_event_listener, after_seconds=1)
    )
    second_sleep_started_at = time.monotonic()
    await ctx.aio_sleep_for(timedelta(seconds=5))
    await reconnect

    return {
        "invocation_count": ctx.invocation_count,
        "second_sleep_seconds": time.monotonic() - second_sleep_started_at,
    }
