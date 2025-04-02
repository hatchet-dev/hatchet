# ❓ Simple

from pydantic import BaseModel

from hatchet_sdk import Context, Hatchet
from hatchet_sdk.runnables.types import EmptyModel

hatchet = Hatchet(debug=True)


class SimpleInput(EmptyModel):
    message: str


class SimpleOutput(BaseModel):
    transformed_message: str

child_task = hatchet.workflow(name="SimpleWorkflow", input=SimpleInput)


@child_task.task(name="step1")
def step1(input: SimpleInput, ctx: Context) -> SimpleOutput:
    print("executed step1: ", input.message)
    return SimpleOutput(transformed_message=input.message.upper())


def main() -> None:
    worker = hatchet.worker("test-worker", slots=1, workflows=[child_task])
    worker.start()


# ‼️

if __name__ == "__main__":
    main()
