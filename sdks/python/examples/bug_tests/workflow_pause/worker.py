import asyncio
from datetime import timedelta

from hatchet_sdk import (
    Context,
    Hatchet,
)

hatchet = Hatchet()


@hatchet.task(
    concurrency=1,
    execution_timeout=timedelta(seconds=3),
    schedule_timeout=timedelta(seconds=15),
)
async def workflow_pause_concurrency_bug_task(_i: None, _c: Context) -> None:
    await asyncio.sleep(1)
