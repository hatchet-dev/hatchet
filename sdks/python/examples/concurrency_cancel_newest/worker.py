import asyncio

from pydantic import BaseModel

from hatchet_sdk import (ConcurrencyExpression, ConcurrencyLimitStrategy,
                         Context, Hatchet)

hatchet = Hatchet(debug=True)


class WorkflowInput(BaseModel):
    group: str


concurrency_cancel_newest_workflow = hatchet.workflow(
    name="ConcurrencyCancelNewest",
    concurrency=ConcurrencyExpression(
        expression="input.group",
        max_runs=1,
        limit_strategy=ConcurrencyLimitStrategy.CANCEL_NEWEST,
    ),
    input_validator=WorkflowInput,
)


@concurrency_cancel_newest_workflow.task()
async def step1(input: WorkflowInput, ctx: Context) -> None:
    for _ in range(50):
        await asyncio.sleep(0.10)


@concurrency_cancel_newest_workflow.task(parents=[step1])
async def step2(input: WorkflowInput, ctx: Context) -> None:
    for _ in range(50):
        await asyncio.sleep(0.10)
