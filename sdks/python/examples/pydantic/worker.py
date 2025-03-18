from datetime import timedelta
from typing import cast

from pydantic import BaseModel

from hatchet_sdk import Context, Hatchet

hatchet = Hatchet(debug=True)


# ❓ Pydantic
# This workflow shows example usage of Pydantic within Hatchet
class ParentInput(BaseModel):
    x: str


class ChildInput(BaseModel):
    a: int
    b: int


parent_workflow = hatchet.workflow(name="Parent", input_validator=ParentInput)
child_workflow = hatchet.workflow(name="Child", input_validator=ChildInput)


@parent_workflow.task(timeout=timedelta(minutes=5))
async def spawn(input: ParentInput, ctx: Context) -> dict[str, str]:
    ## `input` is an instance of `ParentInput`
    print(f"Parent input: {input.x}")

    child = await child_workflow.aio_run(input=ChildInput(a=1, b=10))

    return cast(dict[str, str], await child.aio_result())


class StepResponse(BaseModel):
    status: str


@child_workflow.task()
def process(input: ChildInput, ctx: Context) -> StepResponse:
    ## `input` is an instance of `ChildInput`
    assert input.a + input.b == 11

    return StepResponse(status="success")


@child_workflow.task(parents=[process])
def process2(input: ChildInput, ctx: Context) -> StepResponse:
    ## This is an instance of `StepResponse`
    step_response = ctx.task_output(process)

    assert step_response.status == "success"

    return StepResponse(status="step 2 - success")


def main() -> None:
    worker = hatchet.worker(
        "pydantic-worker", workflows=[parent_workflow, child_workflow]
    )
    worker.start()


# ‼️


if __name__ == "__main__":
    main()
