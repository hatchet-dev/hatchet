from pydantic import BaseModel

from hatchet_sdk import Context, Hatchet


class TaskInput(BaseModel):
    user_id: int


class TaskOutput(BaseModel):
    ok: bool


hatchet = Hatchet()


@hatchet.task(name="externally-triggered-task", input_validator=TaskInput)
async def externally_triggered_task(input: TaskInput, ctx: Context) -> TaskOutput:
    return TaskOutput(ok=True)
