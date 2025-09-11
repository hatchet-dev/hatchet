from pydantic import BaseModel

from hatchet_sdk import Context, EmptyModel, Hatchet

hatchet = Hatchet()


class Input(EmptyModel):
    index: int


@hatchet.task(input_validator=Input)
async def return_exceptions_task(input: Input, ctx: Context) -> dict[str, str]:
    if input.index % 2 == 0:
        raise ValueError(f"error in task with index {input.index}")

    return {"message": "this is a successful task."}
