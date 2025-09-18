from pydantic import BaseModel

from hatchet_sdk import Context, Hatchet, ParentCondition


class Input(BaseModel):
    random_number: int


class StepOutput(BaseModel):
    should_cancel: bool


hatchet = Hatchet(debug=True)

payload_initial_cancel_bug_workflow = hatchet.workflow(
    name="payload-initial-cancel-test",
    input_validator=Input,
)


@payload_initial_cancel_bug_workflow.task()
def step1(input: Input, ctx: Context) -> StepOutput:
    if ctx.retry_count == 0:
        return StepOutput(should_cancel=True)
    else:
        return StepOutput(should_cancel=False)


@payload_initial_cancel_bug_workflow.task(
    parents=[step1],
    cancel_if=[ParentCondition(parent=step1, expression="output.should_cancel")],
)
async def step2(input: Input, ctx: Context) -> StepOutput:
    return StepOutput(should_cancel=False)
