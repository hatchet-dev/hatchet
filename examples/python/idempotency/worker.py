from hatchet_sdk import (
    Context,
    Hatchet,
    TTLBasedIdempotencyConfig,
    StatusBasedIdempotencyConfig,
)
from datetime import timedelta
from pydantic import BaseModel
import asyncio
from typing import Literal

hatchet = Hatchet()

# > idempotency

EVENT_KEY = "idempotency:example"


class IdempotencyInput(BaseModel):
    id: str
    desired_status: Literal["success", "cancel", "fail"] = "success"


@hatchet.task(
    idempotency=TTLBasedIdempotencyConfig(
        key_expression="input.id", ttl=timedelta(minutes=1)
    ),
    input_validator=IdempotencyInput,
    on_events=[EVENT_KEY],
)
async def idempotent_task(input: IdempotencyInput, ctx: Context) -> dict[str, str]:
    return {"result": f"Hello, world from task {input.id}"}




@hatchet.task(
    idempotency=TTLBasedIdempotencyConfig(
        key_expression="input.id", ttl=timedelta(seconds=2)
    ),
    input_validator=IdempotencyInput,
    on_events=[EVENT_KEY],
)
async def idempotent_task_short_window(
    input: IdempotencyInput, ctx: Context
) -> dict[str, str]:
    return {"result": f"Hello, world from task {input.id}"}


# > status_based_idempotency
@hatchet.task(
    idempotency=StatusBasedIdempotencyConfig(
        key_expression="input.id", fallback_ttl=timedelta(seconds=10)
    ),
    input_validator=IdempotencyInput,
)
async def idempotent_status_based_task(
    input: IdempotencyInput,
    ctx: Context,
) -> dict[str, str]:
    if input.desired_status == "success":
        await asyncio.sleep(2)
        return {"result": f"Hello, world from task {input.id}"}

    if input.desired_status == "fail":
        await asyncio.sleep(2)
        raise Exception(f"Task {input.id} failed as requested.")

    if input.desired_status == "cancel":
        await asyncio.sleep(1)
        await ctx.aio_cancel()
        for _ in range(10):
            await asyncio.sleep(1)

    raise Exception(f"Task {input.id} should have been cancelled, but was not.")




def main() -> None:
    worker = hatchet.worker(
        "test-worker",
        workflows=[idempotent_task],
    )
    worker.start()


if __name__ == "__main__":
    main()
