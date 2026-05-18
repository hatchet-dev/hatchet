from hatchet_sdk import Context, Hatchet, EmptyModel
import asyncio
from pydantic import BaseModel

hatchet = Hatchet()


class ChildInput(BaseModel):
    id: int


@hatchet.task(input_validator=ChildInput)
async def child_key_caching_test_child(input: ChildInput, ctx: Context) -> None:
    print("executing child with id", input.id)
    if input.id == 1 and ctx.attempt_number == 1:
        for _ in range(15):
            await asyncio.sleep(1)


@hatchet.task(retries=1)
async def child_key_caching_test_parent(_i: EmptyModel, ctx: Context) -> None:
    children = [1, 2] if ctx.attempt_number == 1 else [2, 3, 1]

    async with asyncio.timeout(3):
        await child_key_caching_test_child.aio_run_many(
            [
                child_key_caching_test_child.create_bulk_run_item(
                    input=ChildInput(id=i),
                    child_key=f"child-{i}",
                )
                for i in children
            ],
        )
