import asyncio

from pydantic import BaseModel

from hatchet_sdk import (
    ConcurrencyExpression,
    ConcurrencyLimitStrategy,
    Context,
    Hatchet,
)


class InputModel(BaseModel):
    concurrency_key: str


hatchet = Hatchet(debug=True)

multiple_concurrent_cancellations_test_workflow = hatchet.workflow(
    name="multiple-concurrent-cancellations-bug-test",
    input_validator=InputModel,
    concurrency=ConcurrencyExpression(
        expression="input.concurrency_key",
        max_runs=1,
        limit_strategy=ConcurrencyLimitStrategy.GROUP_ROUND_ROBIN,
    ),
)


@multiple_concurrent_cancellations_test_workflow.task(execution_timeout="10s")
async def blocking_task(input: InputModel, ctx: Context) -> None:
    await asyncio.sleep(15)


@multiple_concurrent_cancellations_test_workflow.task(parents=[blocking_task])
async def step_2(input: InputModel, ctx: Context) -> None:
    pass


@multiple_concurrent_cancellations_test_workflow.task(parents=[blocking_task])
async def step_3(input: InputModel, ctx: Context) -> None:
    pass
