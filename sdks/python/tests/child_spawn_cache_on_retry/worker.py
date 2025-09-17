import asyncio
from datetime import timedelta

from hatchet_sdk import Context, EmptyModel, Hatchet, TriggerWorkflowOptions

PARENT_EXECUTION_TIMEOUT_SECONDS = 5
PARENT_RETRIES = 2


hatchet = Hatchet(debug=True)


@hatchet.task(
    execution_timeout=timedelta(seconds=PARENT_EXECUTION_TIMEOUT_SECONDS),
    retries=PARENT_RETRIES,
)
async def spawn_cache_on_retry_parent(input: EmptyModel, ctx: Context) -> None:
    await spawn_cache_on_retry_child.aio_run_no_wait(
        options=TriggerWorkflowOptions(
            additional_metadata=ctx.additional_metadata or {}
        )
    )

    for _ in range(60):
        await asyncio.sleep(1)


@hatchet.task()
async def spawn_cache_on_retry_child(input: EmptyModel, ctx: Context) -> None:
    await asyncio.sleep(10)
