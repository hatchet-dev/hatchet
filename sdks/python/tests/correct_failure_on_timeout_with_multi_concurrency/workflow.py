import random

from pydantic import BaseModel

from hatchet_sdk import (
    ConcurrencyExpression,
    ConcurrencyLimitStrategy,
    Context,
    Hatchet,
)


class InputModel(BaseModel):
    concurrency_key: str
    constant: str


hatchet = Hatchet(debug=True)

multiple_concurrent_cancellations_test_workflow = hatchet.workflow(
    name="workflow-bug-test",
    input_validator=InputModel,
    concurrency=ConcurrencyExpression(
        expression="input.concurrency_key",
        max_runs=1,
        limit_strategy=ConcurrencyLimitStrategy.GROUP_ROUND_ROBIN,
    ),
)


@multiple_concurrent_cancellations_test_workflow.task(
    concurrency=[
        ConcurrencyExpression(
            expression="input.concurrency_key",
            max_runs=1,
            limit_strategy=ConcurrencyLimitStrategy.GROUP_ROUND_ROBIN,
        ),
        ConcurrencyExpression(
            expression="input.constant",
            max_runs=1,
            limit_strategy=ConcurrencyLimitStrategy.GROUP_ROUND_ROBIN,
        ),
    ],
)
async def step_1(input: InputModel, ctx: Context) -> None:
    if random.choice([True, False]):
        raise Exception("Error for bug")


@multiple_concurrent_cancellations_test_workflow.task(
    parents=[step_1],
    concurrency=[
        ConcurrencyExpression(
            expression="input.concurrency_key",
            max_runs=1,
            limit_strategy=ConcurrencyLimitStrategy.GROUP_ROUND_ROBIN,
        ),
        ConcurrencyExpression(
            expression="input.constant",
            max_runs=1,
            limit_strategy=ConcurrencyLimitStrategy.GROUP_ROUND_ROBIN,
        ),
    ],
)
async def step_2(input: InputModel, ctx: Context) -> None:
    if random.choice([True, False]):
        raise Exception("Error for bug")

