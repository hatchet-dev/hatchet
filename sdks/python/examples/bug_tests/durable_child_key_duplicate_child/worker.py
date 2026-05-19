from hatchet_sdk import Context, DurableContext, EmptyModel, Hatchet
from pydantic import BaseModel
from typing import Literal
from uuid import uuid4

hatchet = Hatchet()


class Input(BaseModel):
    scenario: Literal["third_unique", "second_unique", "all_duped"]


@hatchet.task()
async def child_child_key_bug(_i: EmptyModel, ctx: Context) -> dict[str, str]:
    return {"id": ctx.workflow_run_id}


@hatchet.durable_task(input_validator=Input)
async def durable_parent_child_key_bug(
    input: Input, ctx: DurableContext
) -> dict[str, str]:
    for i in range(1, 4):
        child_key = ctx.workflow_run_id + f"-child"

        if input.scenario == "third_unique" and i == 3:
            child_key += str(uuid4())

        if input.scenario == "second_unique" and i == 2:
            child_key += str(uuid4())

        await child_child_key_bug.aio_run(child_key=child_key)

    return {"result": "Hello, world!"}
