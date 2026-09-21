from datetime import timedelta
from pydantic import BaseModel

from hatchet_sdk import DurableContext, Hatchet

hatchet = Hatchet()

EVENT_KEY = "scope-repro:shared-key"


class WaiterInput(BaseModel):
    scope: str
    lookback_seconds: int = 1


@hatchet.durable_task(
    name="scope-repro-waiter",
    input_validator=WaiterInput,
    execution_timeout=timedelta(seconds=10),
)
async def scope_waiter(input: WaiterInput, ctx: DurableContext) -> None:
    await ctx.aio_wait_for_event(
        EVENT_KEY,
        scope=input.scope,
        lookback_window=timedelta(seconds=input.lookback_seconds),
    )
