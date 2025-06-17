import asyncio
from datetime import datetime, timedelta, timezone
from subprocess import Popen
from typing import Any

import pytest

from examples.streaming.worker import chunks, stream_task
from hatchet_sdk import Hatchet
from hatchet_sdk.clients.listeners.run_event_listener import (
    StepRunEvent,
    StepRunEventType,
)


async def main() -> None:
    print(len(chunks))
    ref = await stream_task.aio_run_no_wait()

    ix = 0
    streamed_chunks: list[tuple[int, StepRunEvent]] = []

    async for chunk in ref._wrr.stream():
        print(
            f"Received chunk {ix}: {chunk} at {datetime.now(timezone(timedelta(hours=-4), name='EST'))}"
        )
        streamed_chunks.append((ix, chunk))
        ix += 1


if __name__ == "__main__":
    asyncio.run(main())
