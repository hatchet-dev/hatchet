import asyncio

from pydantic import BaseModel

from hatchet_sdk import (
    ConcurrencyStrategy,
    Context,
    Hatchet,
)

hatchet = Hatchet()


# > Cancel In Progress
class WorkflowInput(BaseModel):
    group: str


concurrency_cancel_in_progress_workflow = hatchet.workflow(
    name="ConcurrencyCancelInProgress",
    concurrency=ConcurrencyStrategy(
        expression="input.group",
        max_runs=1,
        strategy="CANCEL_IN_PROGRESS",
    ),
    input_validator=WorkflowInput,
)
# !!


@concurrency_cancel_in_progress_workflow.task()
async def step1(input: WorkflowInput, ctx: Context) -> None:
    for _ in range(50):
        await asyncio.sleep(0.10)


@concurrency_cancel_in_progress_workflow.task(parents=[step1])
async def step2(input: WorkflowInput, ctx: Context) -> None:
    for _ in range(50):
        await asyncio.sleep(0.10)
