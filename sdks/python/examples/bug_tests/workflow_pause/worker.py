from hatchet_sdk import (
    Context,
    EmptyModel,
    Hatchet,
    ConcurrencyExpression,
    ConcurrencyLimitStrategy,
)
import asyncio
from datetime import timedelta

hatchet = Hatchet()


@hatchet.task(
    concurrency=[
        ConcurrencyExpression(
            expression="'*'",
            max_runs=1,
            limit_strategy=ConcurrencyLimitStrategy.GROUP_ROUND_ROBIN,
        )
    ],
    execution_timeout=timedelta(seconds=3),
    schedule_timeout=timedelta(seconds=15),
)
async def workflow_pause_concurrency_bug_task(
    input: EmptyModel,
    ctx: Context,
) -> None:
    await asyncio.sleep(1)
