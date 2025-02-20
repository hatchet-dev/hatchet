from typing import cast

from pydantic import BaseModel

from hatchet_sdk import BaseWorkflow, Context, Hatchet

hatchet = Hatchet(debug=True)


# ❓ Pydantic
# This workflow shows example usage of Pydantic within Hatchet
class ParentInput(BaseModel):
    x: str


class ChildInput(BaseModel):
    a: int
    b: int


parent_workflow = hatchet.declare_workflow(input_validator=ParentInput)
child_workflow = hatchet.declare_workflow(input_validator=ChildInput)


class Parent(BaseWorkflow):
    config = parent_workflow.config

    @hatchet.step(timeout="5m")
    async def spawn(self, context: Context) -> dict[str, str]:
        ## Use `typing.cast` to cast your `workflow_input`
        ## to the type of your `input_validator`
        parent_workflow.get_workflow_input(context)  ## This is a `ParentInput`

        child = await child_workflow.aio_spawn_one(
            ctx=context, input=ChildInput(a=1, b=10)
        )

        return cast(dict[str, str], await child.result())


class StepResponse(BaseModel):
    status: str


class Child(BaseWorkflow):
    config = child_workflow.config

    @hatchet.step()
    def process(self, context: Context) -> StepResponse:
        ## This is an instance `ChildInput`
        child_workflow.get_workflow_input(context)

        return StepResponse(status="success")

    @hatchet.step(parents=["process"])
    def process2(self, context: Context) -> StepResponse:
        ## This is an instance of `StepResponse`
        cast(StepResponse, context.step_output("process"))

        return {"status": "step 2 - success"}  # type: ignore[return-value]

    @hatchet.step(parents=["process2"])
    def process3(self, context: Context) -> StepResponse:
        ## This is an instance of `StepResponse`, even though the
        ## response of `process2` was a dictionary. Note that
        ## Hatchet will attempt to parse that dictionary into
        ## an object of type `StepResponse`
        cast(StepResponse, context.step_output("process2"))

        return StepResponse(status="step 3 - success")


# ‼️


def main() -> None:
    worker = hatchet.worker("pydantic-worker")
    worker.register_workflow(Parent())
    worker.register_workflow(Child())
    worker.start()


if __name__ == "__main__":
    main()
