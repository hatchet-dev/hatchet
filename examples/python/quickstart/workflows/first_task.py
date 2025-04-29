from hatchet_sdk import Context
from pydantic import BaseModel

from ..hatchet_client import hatchet


class SimpleInput(BaseModel):
    message: str


class SimpleOutput(BaseModel):
    transformed_message: str


# Declare the task to run
@hatchet.task(name="first-task")
def first_task(input: SimpleInput, ctx: Context) -> SimpleOutput:
    print("first-task task called")

    return SimpleOutput(transformed_message=input.message.lower())
