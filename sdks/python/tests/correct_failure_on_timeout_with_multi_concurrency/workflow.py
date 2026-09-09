import asyncio
from datetime import timedelta

from pydantic import BaseModel

from hatchet_sdk import (
    ConcurrencyStrategy,
    Context,
    Hatchet,
)

TIMEOUT_SECONDS = 3


class InputModel(BaseModel):
    concurrency_key: str


hatchet = Hatchet()

multiple_concurrent_cancellations_test_workflow = hatchet.workflow(
    name="workflow-bug-test",
    input_validator=InputModel,
    concurrency=ConcurrencyStrategy(
        expression="input.concurrency_key",
        max_runs=1,
        strategy="GROUP_ROUND_ROBIN",
    ),
)


@multiple_concurrent_cancellations_test_workflow.task(
    concurrency=[
        ConcurrencyStrategy(
            expression="input.concurrency_key",
            max_runs=1,
            strategy="GROUP_ROUND_ROBIN",
        ),
    ],
    execution_timeout=timedelta(seconds=TIMEOUT_SECONDS),
)
async def step_1(input: InputModel, ctx: Context) -> None:
    await asyncio.sleep(3)
    raise Exception("Error for bug")


@multiple_concurrent_cancellations_test_workflow.task(
    parents=[step_1],
    concurrency=[
        ConcurrencyStrategy(
            expression="input.concurrency_key",
            max_runs=1,
            strategy="GROUP_ROUND_ROBIN",
        ),
    ],
    execution_timeout=timedelta(seconds=TIMEOUT_SECONDS),
)
async def step_2(input: InputModel, ctx: Context) -> None:
    pass
