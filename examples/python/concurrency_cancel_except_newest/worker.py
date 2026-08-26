import asyncio

from pydantic import BaseModel

from hatchet_sdk import (
    ConcurrencyExpression,
    ConcurrencyLimitStrategy,
    Context,
    Hatchet,
)

hatchet = Hatchet()


# > Cancel Except Newest
class WorkflowInput(BaseModel):
    group: str


concurrency_cancel_except_newest_workflow = hatchet.workflow(
    name="ConcurrencyCancelExceptNewest",
    concurrency=ConcurrencyExpression(
        expression="input.group",
        max_runs=1,
        limit_strategy=ConcurrencyLimitStrategy.CANCEL_EXCEPT_NEWEST,
    ),
    input_validator=WorkflowInput,
)


@concurrency_cancel_except_newest_workflow.task()
async def step1(input: WorkflowInput, ctx: Context) -> None:
    for _ in range(30):
        await asyncio.sleep(0.10)
