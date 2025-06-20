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
    ref = await stream_task.aio_run_no_wait()

    async for chunk in ref._wrr.stream():
        if chunk.type == StepRunEventType.STEP_RUN_EVENT_TYPE_STREAM:
            print(chunk.payload, flush=True, end="")


if __name__ == "__main__":
    asyncio.run(main())
