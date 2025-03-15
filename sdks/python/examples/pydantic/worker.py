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
    child = await child_workflow.aio_run(input=ChildInput(a=1, b=10))

    return cast(dict[str, str], await child.aio_result())


class StepResponse(BaseModel):
    status: str


@child_workflow.task()
def process(input: ChildInput, ctx: Context) -> StepResponse:
    return StepResponse(status="success")


@child_workflow.task(parents=[process])
def process2(input: ChildInput, ctx: Context) -> StepResponse:
    ## This is an instance of `StepResponse`
    ctx.task_output(process)

    return {"status": "step 2 - success"}  # type: ignore[return-value]


@child_workflow.task(parents=[process2])
def process3(input: ChildInput, ctx: Context) -> StepResponse:
    ## This is an instance of `StepResponse`, even though the
    ## response of `process2` was a dictionary. Note that
    ## Hatchet will attempt to parse that dictionary into
    ## an object of type `StepResponse`
    ctx.task_output(process2)

    return StepResponse(status="step 3 - success")


# ‼️


def main() -> None:
    worker = hatchet.worker(
        "pydantic-worker", workflows=[parent_workflow, child_workflow]
    )
    worker.start()


if __name__ == "__main__":
    main()
