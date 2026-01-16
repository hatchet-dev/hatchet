import asyncio
import random
import time

from pydantic import BaseModel

from hatchet_sdk import Context, Hatchet


class MockInput(BaseModel):
    foo: str


class StepOutput(BaseModel):
    random_number: int


class RandomSum(BaseModel):
    sum: int


hatchet = Hatchet(debug=True)

run_detail_test_workflow = hatchet.workflow(
    name="RunDetailTest", input_validator=MockInput
)


@run_detail_test_workflow.task()
async def step1(input: MockInput, ctx: Context) -> StepOutput:
    return StepOutput(random_number=random.randint(1, 100))


@run_detail_test_workflow.task()
async def cancel_step(input: MockInput, ctx: Context) -> None:
    await ctx.aio_cancel()
    for _ in range(10):
        await asyncio.sleep(1)


@run_detail_test_workflow.task()
async def fail_step(input: MockInput, ctx: Context) -> None:
    raise Exception("Intentional Failure")


@run_detail_test_workflow.task()
async def step2(input: MockInput, ctx: Context) -> StepOutput:
    await asyncio.sleep(5)
    return StepOutput(random_number=random.randint(1, 100))


@run_detail_test_workflow.task(parents=[step1, step2])
async def step3(input: MockInput, ctx: Context) -> RandomSum:
    one = ctx.task_output(step1).random_number
    two = ctx.task_output(step2).random_number

    return RandomSum(sum=one + two)


@run_detail_test_workflow.task(parents=[step1, step3])
async def step4(input: MockInput, ctx: Context) -> dict[str, str]:
    print(
        "executed step4",
        time.strftime("%H:%M:%S", time.localtime()),
        input,
        ctx.task_output(step1),
        ctx.task_output(step3),
    )
    return {
        "step4": "step4",
    }
