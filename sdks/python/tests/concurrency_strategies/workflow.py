import asyncio

from pydantic import BaseModel

from hatchet_sdk import (
    ConcurrencyStrategy,
    Context,
    Hatchet,
)


class InputModel(BaseModel):
    key1: str
    key2: str


hatchet = Hatchet()

concurrency_strat = [
    ConcurrencyStrategy(
        expression="input.key1",
        max_runs=1,
        strategy="CANCEL_IN_PROGRESS",
    ),
    ConcurrencyStrategy(
        expression="input.key2",
        max_runs=5,
        strategy="GROUP_ROUND_ROBIN",
    ),
]
concurrency_strategy_workflow = hatchet.workflow(
    name="workflow-concurrency-strategy",
    input_validator=InputModel,
    concurrency=concurrency_strat,
)


@concurrency_strategy_workflow.task()
async def step_1(input: InputModel, ctx: Context) -> None:
    await asyncio.sleep(0.1)


@concurrency_strategy_workflow.task(parents=[step_1])
async def step_2(input: InputModel, ctx: Context) -> None:
    await asyncio.sleep(0.1)


@concurrency_strategy_workflow.task(parents=[step_2])
async def step_3(input: InputModel, ctx: Context) -> None:
    await asyncio.sleep(0.1)


@concurrency_strategy_workflow.task(parents=[step_3])
async def step_4(input: InputModel, ctx: Context) -> None:
    await asyncio.sleep(0.1)
