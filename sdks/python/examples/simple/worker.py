# ❓ Simple

from pydantic import BaseModel

from hatchet_sdk import Context, Hatchet
from hatchet_sdk.runnables.types import EmptyModel

hatchet = Hatchet(debug=True)


class SimpleInput(EmptyModel):
    message: str


class SimpleOutput(BaseModel):
    transformed_message: str


@hatchet.task(name="SimpleWorkflow")
def step1(input: SimpleInput, ctx: Context) -> SimpleOutput:
    print("executed step1: ", input.message)
    return SimpleOutput(transformed_message=input.message.upper())


def main() -> None:
    worker = hatchet.worker("test-worker", slots=1, workflows=[step1])
    worker.start()


# ‼️

if __name__ == "__main__":
    main()
